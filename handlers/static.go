package handlers

import (
	"net/http"

	"github.com/go-chi/chi/v5"
)

func Static(mux chi.Router) {
	fs := http.FileServer(http.Dir("./static"))
	mux.Handle("/static/*", http.StripPrefix("/static/", fs))
}
