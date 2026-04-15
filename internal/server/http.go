package server

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"vpn/internal/peer"
	"vpn/internal/user"
)

func NewHTTP(users *user.Handler, peers *peer.Handler) http.Handler {
	r := chi.NewRouter()

	r.Use(middleware.RequestID)
	r.Use(middleware.Recoverer)
	r.Use(middleware.StripSlashes)

	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	r.Post("/users", users.Create)
	r.Get("/users/{id}", users.GetByID)

	r.Post("/peers", peers.Create)
	r.Get("/peers/{id}", peers.GetByID)

	return r
}
