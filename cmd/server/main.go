// Package main is the entry point to the server. It reads configuration, sets up logging and error handling,
// handles signals from the OS, and starts and stops the server.
package main

import (
	"canvas/jobs"
	"canvas/messaging"
	"canvas/server"
	"canvas/storage"
	"canvas/util"
	"context"
	"fmt"
	"log/slog"
	"net/url"
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

func main() {
	os.Exit(start())
}

func start() int {
	_ = env.Load()
	logEnv := env.GetStringOrDefault("LOG_ENV", "development")
	util.InitializeSlog(logEnv, release)

	host := env.GetStringOrDefault("HOST", "localhost")
	port := env.GetIntOrDefault("PORT", 8080)

	awsConfig, err := config.LoadDefaultConfig(context.Background(),
		config.WithLogger(createAWSLogAdapter()),
		config.WithEndpointResolverWithOptions(createAWSEndpointResolver()),
	)

	if err != nil {
		slog.Info("Error creating AWS config", err)
		return 1
	}
	queue := createQueue(awsConfig)
	s := server.New(server.Options{
		Database:      createDatabase(),
		Host:          host,
		Port:          port,
		Queue:         queue,
		AdminPassword: env.GetStringOrDefault("ADMIN_PASSWORD", "eyDawVH9LLZtaG2q"),
	})
	r := jobs.NewRunner(jobs.NewRunnerOptions{
		Emailer: createEmailer(host, port),
		Queue:   queue,
	})
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGTERM, syscall.SIGINT)
	defer stop()
	eg, ctx := errgroup.WithContext(ctx)

	eg.Go(func() error {
		if err := s.Start(); err != nil {
			slog.Error("Error starting server", err)
			return err
		}
		return nil
	})

	eg.Go(func() error {
		r.Start(ctx)
		return nil
	})

	<-ctx.Done()

	eg.Go(func() error {
		if err := s.Stop(); err != nil {
			slog.Error("Error stopping server", err)
			return err
		}
		return nil
	})

	if err := eg.Wait(); err != nil {

		return 1

	}
	return 0
}

func createDatabase() *storage.Database {
	return storage.NewDatabase(storage.NewDatabaseOptions{
		Host:                  env.GetStringOrDefault("DB_HOST", "localhost"),
		Port:                  env.GetIntOrDefault("DB_PORT", 5432),
		User:                  env.GetStringOrDefault("DB_USER", ""),
		Password:              env.GetStringOrDefault("DB_PASSWORD", ""),
		Name:                  env.GetStringOrDefault("DB_NAME", ""),
		MaxOpenConnections:    env.GetIntOrDefault("DB_MAX_OPEN_CONNECTIONS", 10),
		MaxIdleConnections:    env.GetIntOrDefault("DB_MAX_IDLE_CONNECTIONS", 10),
		ConnectionMaxLifetime: env.GetDurationOrDefault("DB_CONNECTION_MAX_LIFETIME", time.Hour),
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

func createEmailer(host string, port int) *messaging.Emailer {
	baseURL, err := url.Parse(env.GetStringOrDefault(
		"BASE_URL",
		fmt.Sprintf("http://%v:%v", host, port),
	))
	if err != nil {
		slog.Error("BASE_URL is not valid url", "error", err)
	}
	return messaging.NewEmailer(messaging.NewEmailerOptions{
		BaseURL:            baseURL,
		MarketingEmailName: env.GetStringOrDefault("MARKETING_EMAIL_NAME", "Canvas bot"),
		MarketingEmailAddress: env.GetStringOrDefault("MARKETING_EMAIL_ADDRESS",
			"bot@marketing.example.com"),
		Token:                  env.GetStringOrDefault("POSTMARK_TOKEN", ""),
		TransactionalEmailName: env.GetStringOrDefault("TRANSACTIONAL_EMAIL_NAME", "Canvas bot"),
		TransactionalEmailAddress: env.GetStringOrDefault("TRANSACTIONAL_EMAIL_ADDRESS",
			"bot@transactional.example.com"),
	})
}
