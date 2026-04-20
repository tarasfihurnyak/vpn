package server

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"vpn/internal/auth"
	"vpn/internal/peer"
	"vpn/internal/user"
)

func NewHTTP(users *user.Handler, peers *peer.Handler, authHandler *auth.Handler, authMiddleware func(http.Handler) http.Handler, allowedOrigins []string) http.Handler {
	r := chi.NewRouter()

	r.Use(middleware.RequestID)
	r.Use(middleware.Recoverer)
	r.Use(middleware.StripSlashes)

	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	r.Route("/api", func(r chi.Router) {
		r.Route("/auth", func(r chi.Router) {
			r.Post("/login", authHandler.Login)

			r.Group(func(r chi.Router) {
				r.Use(OriginCheck(allowedOrigins))
				r.Post("/refresh", authHandler.Refresh)
				r.Post("/logout", authHandler.Logout)
			})
		})

		// protected routes — require a valid JWT access token
		r.Group(func(r chi.Router) {
			r.Use(authMiddleware)

			r.Post("/users", users.Create)
			r.Get("/users/{id}", users.GetByID)

			r.Post("/peers", peers.Create)
			r.Get("/peers/{id}", peers.GetByID)
		})
	})

	return r
}
