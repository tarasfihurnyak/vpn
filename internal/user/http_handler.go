package user

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/rs/zerolog/log"

	pkghttp "vpn/pkg/http"
)

type Handler struct {
	svc *Service
}

func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

type createRequest struct {
	Username string `json:"username"`
	Email    string `json:"email"`
	Password string `json:"password"`
}

// @Summary      Create user
// @Description  Register a new user account.
// @Tags         users
// @Accept       json
// @Produce      json
// @Param        body  body      createRequest  true  "User details"
// @Success      201   {object}  User
// @Failure      400   {object}  pkghttp.ErrorResponse
// @Failure      500   {object}  pkghttp.ErrorResponse
// @Security     BearerAuth
// @Router       /users [post]
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	var req createRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		var maxErr *http.MaxBytesError
		if errors.As(err, &maxErr) {
			log.Error().Err(err).Msg("create user: request body too large")
			pkghttp.WriteError(w, http.StatusRequestEntityTooLarge, "request body too large")
			return
		}
		log.Error().Err(err).Msg("create user: invalid request body")
		pkghttp.WriteError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Username == "" || req.Email == "" || req.Password == "" {
		log.Error().Msg("create user: missing required fields")
		pkghttp.WriteError(w, http.StatusBadRequest, "username, email and password are required")
		return
	}

	if len(req.Password) < 8 {
		log.Error().Msg("create user: password too short")
		pkghttp.WriteError(w, http.StatusBadRequest, "password must be at least 8 characters")
		return
	}

	u, err := h.svc.Create(r.Context(), req.Username, req.Email, req.Password)
	if err != nil {
		log.Error().Err(err).Str("username", req.Username).Msg("create user failed")
		pkghttp.WriteError(w, http.StatusInternalServerError, "internal server error")
		return
	}

	pkghttp.WriteJSON(w, http.StatusCreated, u)
}

// @Summary      Get user
// @Description  Retrieve a user by ID.
// @Tags         users
// @Produce      json
// @Param        id   path      string  true  "User UUID"
// @Success      200  {object}  User
// @Failure      400  {object}  pkghttp.ErrorResponse
// @Failure      404  {object}  pkghttp.ErrorResponse
// @Failure      500  {object}  pkghttp.ErrorResponse
// @Security     BearerAuth
// @Router       /users/{id} [get]
func (h *Handler) GetByID(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		log.Error().Err(err).Str("id", chi.URLParam(r, "id")).Msg("get user: invalid id")
		pkghttp.WriteError(w, http.StatusBadRequest, "invalid id")
		return
	}

	u, err := h.svc.GetByID(r.Context(), id)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			log.Error().Str("id", id.String()).Msg("get user: not found")
			pkghttp.WriteError(w, http.StatusNotFound, "user not found")
			return
		}
		log.Error().Err(err).Str("id", id.String()).Msg("get user: internal error")
		pkghttp.WriteError(w, http.StatusInternalServerError, "internal server error")
		return
	}

	pkghttp.WriteJSON(w, http.StatusOK, u)
}
