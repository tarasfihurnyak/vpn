package server

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/rs/zerolog/log"

	pkghttp "vpn/pkg/http"
)

// OriginCheck returns a middleware that protects cookie-based endpoints from CSRF.
// It requires the Origin header to be present and match one of the allowed origins.
//
// cookie-driven endpoints are only reached by browsers,
// which always send Origin on cross-site requests.
// Non-browser clients (curl, CLI) use Bearer tokens and never hit these routes.
func OriginCheck(allowed []string) func(http.Handler) http.Handler {
	set := make(map[string]struct{}, len(allowed))
	for _, o := range allowed {
		set[normalizeOrigin(o)] = struct{}{}
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			origin := r.Header.Get("Origin")
			if origin == "" {
				log.Warn().Str("path", r.URL.Path).Msg("csrf: missing Origin header")
				pkghttp.WriteError(w, http.StatusForbidden, "missing origin")
				return
			}

			norm, err := parseOrigin(origin)
			if err != nil {
				log.Warn().Str("origin", origin).Str("path", r.URL.Path).Msg("csrf: invalid origin")
				pkghttp.WriteError(w, http.StatusForbidden, "invalid origin")
				return
			}

			if _, ok := set[norm]; !ok {
				log.Warn().Str("origin", origin).Str("path", r.URL.Path).Msg("csrf: origin not allowed")
				pkghttp.WriteError(w, http.StatusForbidden, "origin not allowed")
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

func parseOrigin(origin string) (string, error) {
	u, err := url.Parse(origin)
	if err != nil || u.Host == "" {
		return "", fmt.Errorf("invalid origin: %q", origin)
	}
	return strings.ToLower(u.Scheme) + "://" + strings.ToLower(u.Host), nil
}

func normalizeOrigin(origin string) string {
	norm, err := parseOrigin(origin)
	if err != nil {
		return strings.ToLower(strings.TrimRight(origin, "/"))
	}
	return norm
}
