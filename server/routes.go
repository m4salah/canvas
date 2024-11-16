package server

import (
	"canvas/handlers"
	"canvas/model"
	"context"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

type signupperMock struct{}

func (s signupperMock) SignupForNewsletter(ctx context.Context, email model.Email) (string, error) {
	return "", nil
}

type confirmerMock struct{}

func (s confirmerMock) ConfirmNewsletterSignup(
	ctx context.Context,
	token string,
) (*model.Email, error) {
	email := model.Email("hello")
	return &email, nil
}

func (s *Server) setupRoutes() {
	s.mux.Use(handlers.AddMetrics(s.metrics))
	handlers.Public(s.mux)
	handlers.Health(s.mux, s.database)
	handlers.Home(s.mux)

	// newsletter routes
	handlers.NewsletterSignup(s.mux, s.database, s.queue)
	handlers.NewsletterThanks(s.mux)
	handlers.NewsletterConfirm(s.mux, s.database, s.queue)
	handlers.NewsletterConfirmed(s.mux)

	// Admin routes
	s.mux.Group(func(r chi.Router) {
		r.Use(middleware.BasicAuth("canvas", map[string]string{"admin": s.adminPassword}))

		handlers.MigrateTo(r, s.database)
		handlers.MigrateUp(r, s.database)
	})

	metricsAuth := middleware.BasicAuth(
		"metrics",
		map[string]string{"prometheus": s.metricsPassword},
	)
	handlers.Metrics(s.mux.With(metricsAuth), s.metrics)
}
