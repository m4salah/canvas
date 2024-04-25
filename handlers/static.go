package handlers

import (
	"net/http"

	"github.com/go-chi/chi/v5"
)

func Public(mux chi.Router) {
	fs := http.FileServer(http.Dir("./public"))
	mux.Handle("/public/*", http.StripPrefix("/public/", fs))
}
