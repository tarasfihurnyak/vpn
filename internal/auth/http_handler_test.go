package auth_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"

	"vpn/internal/auth"
	"vpn/internal/testutil"
)

func loginAndGetCookie(t *testing.T, h *auth.Handler, ctx context.Context, login, password string) *http.Cookie {
	t.Helper()
	req, rec := testutil.NewJSONRequest(http.MethodPost, "/auth/login",
		`{"login":"`+login+`","password":"`+password+`"}`)
	req = req.WithContext(ctx)
	h.Login(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)
	for _, c := range rec.Result().Cookies() {
		if c.Name == "refresh_token" {
			return c
		}
	}
	t.Fatal("refresh_token cookie not found")
	return nil
}

func TestAuthHandler_Login(t *testing.T) {
	svc, userSvc, ctx := newDeps(t)
	h := auth.NewHandler(svc, false)

	_, err := userSvc.Create(ctx, "huser", "huser@example.com", "password123")
	require.NoError(t, err)

	tests := []struct {
		name         string
		body         string
		bodyLimit    int64 // 0 means no limit
		expectedCode int
	}{
		{
			name:         "ok",
			body:         `{"login":"huser","password":"password123"}`,
			expectedCode: http.StatusOK,
		},
		{
			name:         "invalid credentials",
			body:         `{"login":"huser","password":"wrongpassword"}`,
			expectedCode: http.StatusUnauthorized,
		},
		{
			name:         "missing password",
			body:         `{"login":"huser"}`,
			expectedCode: http.StatusBadRequest,
		},
		{
			name:         "invalid json",
			body:         `{bad}`,
			expectedCode: http.StatusBadRequest,
		},
		{
			name:         "body too large",
			body:         `{"login":"huser","password":"password123"}`,
			bodyLimit:    1,
			expectedCode: http.StatusRequestEntityTooLarge,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			req, rec := testutil.NewJSONRequest(http.MethodPost, "/auth/login", tc.body)
			req = req.WithContext(ctx)
			if tc.bodyLimit > 0 {
				req.Body = http.MaxBytesReader(rec, req.Body, tc.bodyLimit)
			}
			h.Login(rec, req)

			require.Equal(t, tc.expectedCode, rec.Code)
			require.Equal(t, "application/json", rec.Header().Get("Content-Type"))
			if tc.expectedCode != http.StatusOK {
				return
			}

			var pair auth.TokenPair
			require.NoError(t, json.NewDecoder(rec.Body).Decode(&pair))
			require.NotEmpty(t, pair.AccessToken)

			var found bool
			for _, c := range rec.Result().Cookies() {
				if c.Name == "refresh_token" {
					found = true
					require.True(t, c.HttpOnly)
					require.NotEmpty(t, c.Value)
				}
			}
			require.True(t, found, "refresh_token cookie should be set")
		})
	}
}

func TestAuthHandler_Refresh(t *testing.T) {
	svc, userSvc, ctx := newDeps(t)
	h := auth.NewHandler(svc, false)

	_, err := userSvc.Create(ctx, "ruser", "ruser@example.com", "password123")
	require.NoError(t, err)

	validCookie := loginAndGetCookie(t, h, ctx, "ruser", "password123")

	tests := []struct {
		name         string
		cookie       *http.Cookie
		expectedCode int
	}{
		{
			name:         "ok",
			cookie:       validCookie,
			expectedCode: http.StatusOK,
		},
		{
			name:         "missing cookie",
			cookie:       nil,
			expectedCode: http.StatusUnauthorized,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/auth/refresh", nil).WithContext(ctx)
			if tc.cookie != nil {
				req.AddCookie(tc.cookie)
			}
			rec := httptest.NewRecorder()
			h.Refresh(rec, req)

			require.Equal(t, tc.expectedCode, rec.Code)
			if tc.expectedCode != http.StatusOK {
				return
			}

			var pair auth.TokenPair
			require.NoError(t, json.NewDecoder(rec.Body).Decode(&pair))
			require.NotEmpty(t, pair.AccessToken)
		})
	}
}

func TestAuthHandler_Logout(t *testing.T) {
	svc, userSvc, ctx := newDeps(t)
	h := auth.NewHandler(svc, false)

	_, err := userSvc.Create(ctx, "luser", "luser@example.com", "password123")
	require.NoError(t, err)

	validCookie := loginAndGetCookie(t, h, ctx, "luser", "password123")

	tests := []struct {
		name         string
		cookie       *http.Cookie
		expectedCode int
	}{
		{
			name:         "ok - token revoked after logout",
			cookie:       validCookie,
			expectedCode: http.StatusNoContent,
		},
		{
			name:         "missing cookie",
			cookie:       nil,
			expectedCode: http.StatusNoContent,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/auth/logout", nil).WithContext(ctx)
			if tc.cookie != nil {
				req.AddCookie(tc.cookie)
			}
			rec := httptest.NewRecorder()
			h.Logout(rec, req)

			require.Equal(t, tc.expectedCode, rec.Code)
			if tc.cookie != nil {
				req := httptest.NewRequest(http.MethodPost, "/auth/refresh", nil).WithContext(ctx)
				req.AddCookie(tc.cookie)
				rec := httptest.NewRecorder()
				h.Refresh(rec, req)
				require.Equal(t, http.StatusUnauthorized, rec.Code)
			}
		})
	}
}

func TestAuthHandler_Middleware(t *testing.T) {
	svc, userSvc, ctx := newDeps(t)
	h := auth.NewHandler(svc, false)

	_, err := userSvc.Create(ctx, "mwuser", "mwuser@example.com", "password123")
	require.NoError(t, err)

	loginReq, loginRec := testutil.NewJSONRequest(http.MethodPost, "/auth/login",
		`{"login":"mwuser","password":"password123"}`)
	loginReq = loginReq.WithContext(ctx)
	h.Login(loginRec, loginReq)
	require.Equal(t, http.StatusOK, loginRec.Code)
	var pair auth.TokenPair
	require.NoError(t, json.NewDecoder(loginRec.Body).Decode(&pair))

	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, ok := auth.UserIDFromContext(r.Context())
		if ok {
			w.WriteHeader(http.StatusOK)
		} else {
			w.WriteHeader(http.StatusInternalServerError)
		}
	})

	tests := []struct {
		name         string
		authHeader   string
		expectedCode int
	}{
		{
			name:         "valid token",
			authHeader:   "Bearer " + pair.AccessToken,
			expectedCode: http.StatusOK,
		},
		{
			name:         "no token",
			authHeader:   "",
			expectedCode: http.StatusUnauthorized,
		},
		{
			name:         "invalid token",
			authHeader:   "Bearer invalid.token.here",
			expectedCode: http.StatusUnauthorized,
		},
		{
			name:         "wrong scheme",
			authHeader:   "Basic " + pair.AccessToken,
			expectedCode: http.StatusUnauthorized,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/users/me", nil)
			if tc.authHeader != "" {
				req.Header.Set("Authorization", tc.authHeader)
			}
			rec := httptest.NewRecorder()
			svc.Middleware(next).ServeHTTP(rec, req)

			require.Equal(t, tc.expectedCode, rec.Code)
		})
	}
}
