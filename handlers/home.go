package handlers

import (
	"net/http"

	"canvas/views"

	"github.com/go-chi/chi/v5"
)

func Home(mux chi.Router) {
	mux.Get("/", func(w http.ResponseWriter, r *http.Request) {
		views.Page().Render(r.Context(), w)
	})
}
