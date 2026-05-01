package server

import (
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"golang.org/x/time/rate"

	"vpn/docs"
	"vpn/internal/auth"
	"vpn/internal/peer"
	"vpn/internal/user"
)

func NewHTTP(users *user.Handler, peers *peer.Handler, authHandler *auth.Handler, authMiddleware func(http.Handler) http.Handler, allowedOrigins []string) http.Handler {
	r := chi.NewRouter()

	// 5 req/min sustained, burst of 3 — applied per IP
	loginLimiter := NewRateLimiter(rate.Every(time.Minute/5), 3)

	r.Use(middleware.RequestID)
	r.Use(RequestLogger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.StripSlashes)
	r.Use(MaxBodySize(1 << 20)) // 1 MB
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   allowedOrigins,
		AllowedMethods:   []string{"GET", "POST", "OPTIONS"},
		AllowedHeaders:   []string{"Authorization", "Content-Type", "Origin"},
		AllowCredentials: true,
	}))

	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	r.Get("/docs/swagger.json", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(docs.SwaggerJSON)
	})

	r.Get("/docs", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write([]byte(`<!doctype html>
<html>
  <head>
    <title>VPN API Reference</title>
    <meta charset="utf-8" />
    <meta name="viewport" content="width=device-width, initial-scale=1" />
  </head>
  <body>
    <div id="app"></div>
    <script src="https://cdn.jsdelivr.net/npm/@scalar/api-reference"></script>
    <script>
      Scalar.createApiReference('#app', {
        url: '/docs/swagger.json',
      })
    </script>
  </body>
</html>`))
	})

	r.Route("/api", func(r chi.Router) {
		r.Route("/auth", func(r chi.Router) {
			r.With(loginLimiter.Middleware).Post("/login", authHandler.Login)

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
