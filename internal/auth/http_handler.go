package auth

import (
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/rs/zerolog/log"

	pkghttp "vpn/pkg/http"
)

const (
	refreshCookieName = "refresh_token"
	refreshCookiePath = "/api/auth"
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
	Login    string `json:"login"    example:"admin@gmail.com"` // username or email
	Password string `json:"password" example:"ChangeMe123!"`
}

// Login authenticates a user and returns an access token + sets a refresh cookie.
//
// @Summary      Login
// @Description  Authenticate with username/email and password. Returns an access token and sets an httpOnly refresh cookie.
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        body  body      loginRequest  true  "Credentials"
// @Success      200   {object}  TokenPair
// @Failure      400   {object}  pkghttp.ErrorResponse
// @Failure      401   {object}  pkghttp.ErrorResponse
// @Failure      429   {object}  pkghttp.ErrorResponse
// @Failure      500   {object}  pkghttp.ErrorResponse
// @Router       /auth/login [post]
func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	var req loginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		var maxErr *http.MaxBytesError
		if errors.As(err, &maxErr) {
			log.Error().Err(err).Msg("login: request body too large")
			pkghttp.WriteError(w, http.StatusRequestEntityTooLarge, "request body too large")
			return
		}
		log.Error().Err(err).Msg("login: invalid request body")
		pkghttp.WriteError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Login == "" || req.Password == "" {
		log.Error().Msg("login: missing login or password")
		pkghttp.WriteError(w, http.StatusBadRequest, "login and password are required")
		return
	}

	pair, rawRefresh, err := h.svc.Login(r.Context(), req.Login, req.Password)
	if err != nil {
		if errors.Is(err, ErrInvalidCredentials) {
			log.Error().Msg("login: invalid credentials")
			pkghttp.WriteError(w, http.StatusUnauthorized, "invalid credentials")
			return
		}
		log.Error().Err(err).Msg("login: internal error")
		pkghttp.WriteError(w, http.StatusInternalServerError, "internal server error")
		return
	}

	h.setRefreshCookie(w, rawRefresh, time.Now().Add(h.svc.refreshTTL))
	pkghttp.WriteJSON(w, http.StatusOK, pair)
}

// Refresh reads the httpOnly refresh cookie, validates it (with rotation), and
// issues a new token pair.
//
// @Summary      Refresh tokens
// @Description  Rotate the refresh token (read from httpOnly cookie) and return a new token pair.
// @Description  **Requires `Origin` header** matching an allowed origin (CSRF protection). Example: `Origin: https://localhost`.
// @Tags         auth
// @Produce      json
// @Param        Origin  header    string  true  "Allowed origin for CSRF protection, e.g. https://localhost"
// @Success      200  {object}  TokenPair
// @Failure      401  {object}  pkghttp.ErrorResponse
// @Failure      403  {object}  pkghttp.ErrorResponse  "missing or disallowed Origin header"
// @Failure      500  {object}  pkghttp.ErrorResponse
// @Router       /auth/refresh [post]
func (h *Handler) Refresh(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie(refreshCookieName)
	if err != nil {
		log.Error().Msg("refresh: missing refresh token cookie")
		pkghttp.WriteError(w, http.StatusUnauthorized, "missing refresh token")
		return
	}

	pair, rawRefresh, err := h.svc.Refresh(r.Context(), cookie.Value)
	if err != nil {
		h.clearRefreshCookie(w)
		if errors.Is(err, ErrInvalidToken) ||
			errors.Is(err, ErrTokenRevoked) ||
			errors.Is(err, ErrTokenExpired) {
			log.Error().Err(err).Msg("refresh: invalid or expired token")
			pkghttp.WriteError(w, http.StatusUnauthorized, "invalid or expired refresh token")
			return
		}
		log.Error().Err(err).Msg("refresh: internal error")
		pkghttp.WriteError(w, http.StatusInternalServerError, "internal server error")
		return
	}

	h.setRefreshCookie(w, rawRefresh, time.Now().Add(h.svc.refreshTTL))
	pkghttp.WriteJSON(w, http.StatusOK, pair)
}

// Logout revokes the refresh token and clears the cookie.
//
// @Summary      Logout
// @Description  Revoke the refresh token and clear the httpOnly cookie.
// @Description  **Requires `Origin` header** matching an allowed origin (CSRF protection). Example: `Origin: https://localhost`.
// @Tags         auth
// @Param        Origin  header    string  true  "Allowed origin for CSRF protection, e.g. https://localhost"
// @Success      204
// @Failure      403  {object}  pkghttp.ErrorResponse  "missing or disallowed Origin header"
// @Router       /auth/logout [post]
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
