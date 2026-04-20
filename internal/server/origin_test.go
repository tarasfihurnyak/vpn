package server_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"

	"vpn/internal/server"
)

func okHandler(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) }

func TestOriginCheck(t *testing.T) {
	allowed := []string{"http://localhost:5173", "HTTPS://Admin.Example.Com"}
	mw := server.OriginCheck(allowed)
	h := mw(http.HandlerFunc(okHandler))

	tests := []struct {
		name   string
		method string
		origin string
		want   int
	}{
		// allowed origins
		{"allowed http", http.MethodPost, "http://localhost:5173", http.StatusOK},
		{"allowed https uppercase", http.MethodPost, "https://admin.example.com", http.StatusOK},
		{"allowed with trailing slash", http.MethodPost, "https://admin.example.com/", http.StatusOK},

		// blocked origins
		{"unknown origin", http.MethodPost, "https://evil.com", http.StatusForbidden},
		{"lookalike with port", http.MethodPost, "https://admin.example.com:443", http.StatusForbidden},

		// missing origin
		{"missing origin POST", http.MethodPost, "", http.StatusForbidden},
		{"missing origin DELETE", http.MethodDelete, "", http.StatusForbidden},

		// malformed origin
		{"malformed origin", http.MethodPost, "not-a-url", http.StatusForbidden},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(tc.method, "/auth/refresh", nil)
			if tc.origin != "" {
				req.Header.Set("Origin", tc.origin)
			}
			rec := httptest.NewRecorder()
			h.ServeHTTP(rec, req)
			require.Equal(t, tc.want, rec.Code)
		})
	}
}
