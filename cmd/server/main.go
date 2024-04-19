// Package main is the entry point to the server. It reads configuration, sets up logging and error handling,
// handles signals from the OS, and starts and stops the server.
package main

import (
	"canvas/server"
	"canvas/storage"
	"canvas/util"
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

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

	s := server.New(server.Options{
		Database: createDatabase(),
		Host:     host,
		Port:     port,
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
