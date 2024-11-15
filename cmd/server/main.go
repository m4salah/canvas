// Package main is the entry point to the server. It reads configuration, sets up logging and error handling,
// handles signals from the OS, and starts and stops the server.
package main

import (
	"canvas/jobs"
	"canvas/messaging"
	"canvas/server"
	"canvas/storage"
	"canvas/types"
	"canvas/util"
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/aws/smithy-go/logging"
	"github.com/maragudk/env"
	"golang.org/x/sync/errgroup"
)

// release is set through the linker at build time, generally from a git SHA.
// Used for logging and error reporting.
var release string

var envConfig types.Config

func main() {
	if err := util.LoadConfig(&envConfig); err != nil {
		panic(err)
	}
	util.InitializeSlog(envConfig.LogEnv, release)
	os.Exit(start())
}

func start() int {
	awsConfig, err := config.LoadDefaultConfig(context.Background(),
		config.WithLogger(createAWSLogAdapter()),
		config.WithEndpointResolverWithOptions(createAWSEndpointResolver()),
	)

	if err != nil {
		slog.Error("Error creating AWS config", util.ErrAttr(err))
		return 1
	}

	// create a new queue
	queue := createQueue(awsConfig)

	// create the server
	s := server.New(server.Options{
		Database:      createDatabase(),
		Host:          envConfig.Host,
		Port:          envConfig.Port,
		Queue:         queue,
		AdminPassword: envConfig.AdminPassword,
	})

	// create the jobs runner
	r := jobs.NewRunner(jobs.NewRunnerOptions{
		Emailer: createEmailer(),
		Queue:   queue,
	})
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGTERM, syscall.SIGINT)
	defer stop()
	eg, ctx := errgroup.WithContext(ctx)

	// spawn the server on it's goroutine
	eg.Go(func() error {
		if err := s.Start(); err != nil {
			slog.Error("Error starting server", util.ErrAttr(err))
			return err
		}
		return nil
	})

	// spawn the job runner in a another goroutine
	eg.Go(func() error {
		r.Start(ctx)
		return nil
	})

	<-ctx.Done()

	// gracefully shutdown the server
	eg.Go(func() error {
		if err := s.Stop(); err != nil {
			slog.Error("Error stopping server", util.ErrAttr(err))
			return err
		}
		return nil
	})

	// wait on all goroutine in error group to finish
	// and if there is an error exit with non zero status (fail)
	if err := eg.Wait(); err != nil {
		return 1
	}

	// otherwise return zero status (success)
	return 0
}

func createDatabase() *storage.Database {
	return storage.NewDatabase(storage.NewDatabaseOptions{
		Host:                  envConfig.DBHost,
		Port:                  envConfig.DBPort,
		User:                  envConfig.DBUser,
		Password:              envConfig.DBPassword,
		Name:                  envConfig.DBName,
		MaxOpenConnections:    envConfig.DBMaxOpenConnections,
		MaxIdleConnections:    envConfig.DBMaxIdleConnections,
		ConnectionMaxLifetime: envConfig.DBConnectionMaxLifetime,
	})
}

func createAWSLogAdapter() logging.LoggerFunc {
	return func(classification logging.Classification, format string, v ...interface{}) {
		switch classification {
		case logging.Debug:
			slog.Debug(format, slog.Any("attr", v))
		case logging.Warn:
			slog.Warn(format, slog.Any("attr", v))
		}
	}
}

// createAWSEndpointResolver used for local development endpoints.
// See https://aws.github.io/aws-sdk-go-v2/docs/configuring-sdk/endpoints/
func createAWSEndpointResolver() aws.EndpointResolverWithOptionsFunc {
	sqsEndpointURL := env.GetStringOrDefault("SQS_ENDPOINT_URL", "")

	return func(service, region string, opts ...any) (aws.Endpoint, error) {
		if sqsEndpointURL != "" && service == sqs.ServiceID {
			return aws.Endpoint{
				URL: sqsEndpointURL,
			}, nil
		}
		// Fallback to default endpoint
		return aws.Endpoint{}, &aws.EndpointNotFoundError{}
	}
}

// …

func createQueue(awsConfig aws.Config) *messaging.Queue {
	return messaging.NewQueue(messaging.NewQueueOptions{
		Config:   awsConfig,
		Name:     env.GetStringOrDefault("QUEUE_NAME", "jobs"),
		WaitTime: env.GetDurationOrDefault("QUEUE_WAIT_TIME", 20*time.Second),
	})
}

func createEmailer() *messaging.Emailer {
	return messaging.NewEmailer(messaging.NewEmailerOptions{
		BaseURL:            &envConfig.BaseURL,
		MarketingEmailName: env.GetStringOrDefault("MARKETING_EMAIL_NAME", "Canvas bot"),
		MarketingEmailAddress: env.GetStringOrDefault("MARKETING_EMAIL_ADDRESS",
			"bot@marketing.example.com"),
		Token:                  env.GetStringOrDefault("POSTMARK_TOKEN", ""),
		TransactionalEmailName: env.GetStringOrDefault("TRANSACTIONAL_EMAIL_NAME", "Canvas bot"),
		TransactionalEmailAddress: env.GetStringOrDefault("TRANSACTIONAL_EMAIL_ADDRESS",
			"bot@transactional.example.com"),
	})
}
