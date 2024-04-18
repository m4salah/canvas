// Package main is the entry point to the server. It reads configuration, sets up logging and error handling,
// handles signals from the OS, and starts and stops the server.
package main

import (
	"canvas/server"
	"canvas/util"
	"context"
	"log/slog"
	"os"
	"os/signal"
	"strconv"
	"syscall"

	"golang.org/x/sync/errgroup"
)

// release is set through the linker at build time, generally from a git SHA.
// Used for logging and error reporting.
var release string

func main() {
	os.Exit(start())
}

func start() int {
	logEnv := getStringOrDefault("LOG_ENV", "development")
	util.InitializeSlog(logEnv, release)
	host := getStringOrDefault("HOST", "localhost")
	port := getIntOrDefault("PORT", 8080)

	s := server.New(server.Options{
		Host: host,
		Port: port,
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

func getStringOrDefault(name, defaultV string) string {
	v, ok := os.LookupEnv(name)
	if !ok {
		return defaultV
	}
	return v
}

func getIntOrDefault(name string, defaultV int) int {

	v, ok := os.LookupEnv(name)
	if !ok {
		return defaultV
	}
	vAsInt, err := strconv.Atoi(v)
	if err != nil {
		return defaultV
	}
	return vAsInt

}
