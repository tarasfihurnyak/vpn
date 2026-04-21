package server

import "net/http"

// MaxBodySize returns a middleware that limits the size of request bodies.
// If the body exceeds limit bytes, subsequent reads will return an error
// that wraps *http.MaxBytesError, causing handlers to respond with 413.
func MaxBodySize(limit int64) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			r.Body = http.MaxBytesReader(w, r.Body, limit)
			next.ServeHTTP(w, r)
		})
	}
}
