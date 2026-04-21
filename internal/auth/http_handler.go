package auth

import (
	"encoding/json"
	"errors"
	"net/http"
	"time"

	pkghttp "vpn/pkg/http"
)

const (
	refreshCookieName = "refresh_token"
	refreshCookiePath = "/auth"
)

// Handler handles auth HTTP endpoints.
type Handler struct {
	svc          *Service
	secureCookie bool
}

func NewHandler(svc *Service, secureCookie bool) *Handler {
	return &Handler{svc: svc, secureCookie: secureCookie}
}

type loginRequest struct {
	Login    string `json:"login"` // username or email
	Password string `json:"password"`
}

// Login authenticates a user and returns an access token + sets a refresh cookie.
func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	var req loginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		var maxErr *http.MaxBytesError
		if errors.As(err, &maxErr) {
			pkghttp.WriteError(w, http.StatusRequestEntityTooLarge, "request body too large")
			return
		}
		pkghttp.WriteError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Login == "" || req.Password == "" {
		pkghttp.WriteError(w, http.StatusBadRequest, "login and password are required")
		return
	}

	pair, rawRefresh, err := h.svc.Login(r.Context(), req.Login, req.Password)
	if err != nil {
		if errors.Is(err, ErrInvalidCredentials) {
			pkghttp.WriteError(w, http.StatusUnauthorized, "invalid credentials")
			return
		}
		pkghttp.WriteError(w, http.StatusInternalServerError, "internal server error")
		return
	}

	h.setRefreshCookie(w, rawRefresh, time.Now().Add(h.svc.refreshTTL))
	pkghttp.WriteJSON(w, http.StatusOK, pair)
}

// Refresh reads the httpOnly refresh cookie, validates it (with rotation), and
// issues a new token pair.
func (h *Handler) Refresh(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie(refreshCookieName)
	if err != nil {
		pkghttp.WriteError(w, http.StatusUnauthorized, "missing refresh token")
		return
	}

	pair, rawRefresh, err := h.svc.Refresh(r.Context(), cookie.Value)
	if err != nil {
		h.clearRefreshCookie(w)
		if errors.Is(err, ErrInvalidToken) ||
			errors.Is(err, ErrTokenRevoked) ||
			errors.Is(err, ErrTokenExpired) {
			pkghttp.WriteError(w, http.StatusUnauthorized, "invalid or expired refresh token")
			return
		}
		pkghttp.WriteError(w, http.StatusInternalServerError, "internal server error")
		return
	}

	h.setRefreshCookie(w, rawRefresh, time.Now().Add(h.svc.refreshTTL))
	pkghttp.WriteJSON(w, http.StatusOK, pair)
}

// Logout revokes the refresh token and clears the cookie.
func (h *Handler) Logout(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie(refreshCookieName)
	if err == nil {
		_ = h.svc.Logout(r.Context(), cookie.Value)
		h.clearRefreshCookie(w)
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) setRefreshCookie(w http.ResponseWriter, token string, expires time.Time) {
	http.SetCookie(w, &http.Cookie{
		Name:     refreshCookieName,
		Value:    token,
		HttpOnly: true,
		Secure:   h.secureCookie,
		SameSite: http.SameSiteStrictMode,
		Path:     refreshCookiePath,
		Expires:  expires,
	})
}

func (h *Handler) clearRefreshCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     refreshCookieName,
		Value:    "",
		HttpOnly: true,
		Secure:   h.secureCookie,
		SameSite: http.SameSiteStrictMode,
		Path:     refreshCookiePath,
		MaxAge:   -1,
	})
}
