// Package server contains everything for setting up and running the HTTP server.
package server

import (
	"canvas/messaging"
	"canvas/storage"
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"

	"net/http"

	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
)

type Server struct {
	address       string
	adminPassword string
	database      *storage.Database
	mux           chi.Router
	queue         *messaging.Queue
	server        *http.Server
}

type Options struct {
	Database      *storage.Database
	Host          string
	Port          int
	Queue         *messaging.Queue
	AdminPassword string
}

func New(opts Options) *Server {
	address := net.JoinHostPort(opts.Host, strconv.Itoa(opts.Port))
	mux := chi.NewMux()
	return &Server{
		address:       address,
		mux:           mux,
		database:      opts.Database,
		adminPassword: opts.AdminPassword,
		queue:         opts.Queue,
		server: &http.Server{
			Addr:              address,
			Handler:           mux,
			ReadTimeout:       5 * time.Second,
			ReadHeaderTimeout: 5 * time.Second,
			WriteTimeout:      5 * time.Second,

			IdleTimeout: 5 * time.Second,
		},
	}
}

// Start the Server by setting up routes and listening for HTTP requests on the given address.

func (s *Server) Start() error {
	if err := s.database.Connect(); err != nil {
		return fmt.Errorf("error connecting to database: %w", err)
	}
	s.setupRoutes()

	slog.Info("Starting the server", slog.String("address", s.address))
	if err := s.server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return fmt.Errorf("error starting server: %w", err)
	}
	return nil
}

// Stop the Server gracefully within the timeout.
func (s *Server) Stop() error {
	slog.Info("Stopping")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := s.server.Shutdown(ctx); err != nil {
		return fmt.Errorf("error stopping server: %w", err)
	}

	return nil
}
