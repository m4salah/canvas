package handlers

import (
	"context"
	"net/http"

	"github.com/go-chi/chi/v5"

	"canvas/model"
	"canvas/views"
)

// signupper interface
type signupper interface {
	SignupForNewsletter(ctx context.Context, email model.Email) (string, error)
}

// sender interface
type sender interface {
	Send(ctx context.Context, m model.Message) error
}

func NewsletterSignup(mux chi.Router, s signupper, q sender) {
	mux.Post("/newsletter/signup", func(w http.ResponseWriter, r *http.Request) {
		email := model.Email(r.FormValue("email"))

		if !email.IsValid() {
			http.Error(w, "email is invalid", http.StatusBadRequest)
			return
		}

		token, err := s.SignupForNewsletter(r.Context(), email)
		if err != nil {
			http.Error(w, "error signing up, refresh to try again", http.StatusBadGateway)
			return
		}

		// send the message to the job queue.
		err = q.Send(r.Context(), model.Message{
			"job":   "confirmation_email",
			"email": email.String(),
			"token": token,
		})

		if err != nil {
			http.Error(w, "error signing up, refresh to try again", http.StatusBadGateway)
			return
		}

		http.Redirect(w, r, "/newsletter/thanks", http.StatusFound)
	})
}

func NewsletterThanks(mux chi.Router) {
	mux.Get("/newsletter/thanks", func(w http.ResponseWriter, r *http.Request) {
		_ = views.NewsletterThanksPage("/newsletter/thanks").Render(w)
	})
}

type confirmer interface {
	ConfirmNewsletterSignup(ctx context.Context, token string) (*model.Email, error)
}

func NewsletterConfirm(mux chi.Router, s confirmer, q sender) {
	mux.Get("/newsletter/confirm", func(w http.ResponseWriter, r *http.Request) {
		token := r.FormValue("token")

		_ = views.NewsletterConfirmPage("/newsletter/confirm", token).Render(w)
	})

	mux.Post("/newsletter/confirm", func(w http.ResponseWriter, r *http.Request) {
		token := r.FormValue("token")

		email, err := s.ConfirmNewsletterSignup(r.Context(), token)
		if err != nil {
			http.Error(
				w,
				"error saving email address confirmation, refresh to try again",
				http.StatusBadGateway,
			)
			return
		}
		if email == nil {
			http.Error(w, "bad token", http.StatusBadRequest)
			return
		}

		err = q.Send(r.Context(), model.Message{
			"job":   "welcome_email",
			"email": email.String(),
		})
		if err != nil {
			http.Error(
				w,
				"error saving email address confirmation, refresh to try again",
				http.StatusBadGateway,
			)
			return
		}

		http.Redirect(w, r, "/newsletter/confirmed", http.StatusFound)
	})
}

func NewsletterConfirmed(mux chi.Router) {
	mux.Get("/newsletter/confirmed", func(w http.ResponseWriter, r *http.Request) {
		_ = views.NewsletterConfirmedPage("/newsletter/confirmed").Render(w)
	})
}
