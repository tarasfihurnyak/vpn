package auth

import (
	"context"
	"net/http"
	"strings"

	pkghttp "vpn/pkg/http"
)

// Middleware validates the Bearer JWT in the Authorization header and
// injects the user ID into the request context.
func (s *Service) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			pkghttp.WriteError(w, http.StatusUnauthorized, "missing authorization header")
			return
		}

		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || !strings.EqualFold(parts[0], "bearer") {
			pkghttp.WriteError(w, http.StatusUnauthorized, "invalid authorization header format")
			return
		}

		userID, err := s.ValidateAccessToken(parts[1])
		if err != nil {
			pkghttp.WriteError(w, http.StatusUnauthorized, "invalid or expired token")
			return
		}

		ctx := context.WithValue(r.Context(), userIDKey, userID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
