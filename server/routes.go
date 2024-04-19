package server

import (
	"canvas/handlers"
	"canvas/model"
	"context"
)

type signupperMock struct{}

func (s signupperMock) SignupForNewsletter(ctx context.Context, email model.Email) (string, error) {
	return "", nil
}

func (s *Server) setupRoutes() {
	handlers.Static(s.mux)
	handlers.Health(s.mux)
	handlers.Home(s.mux)

	// newsletter routes
	handlers.NewsletterSignup(s.mux, signupperMock{})
	handlers.NewsletterThanks(s.mux)
}
