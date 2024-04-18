package handlers

import (
	"github.com/go-chi/chi/v5"
	"net/http"
)

func Health(mux chi.Router) {
	mux.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		// 🤪
	})
}
