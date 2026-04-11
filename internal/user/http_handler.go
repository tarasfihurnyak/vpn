package user

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgtype"

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
}

func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	var req createRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		pkghttp.WriteError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Username == "" || req.Email == "" {
		pkghttp.WriteError(w, http.StatusBadRequest, "username and email are required")
		return
	}

	u, err := h.svc.Create(r.Context(), req.Username, req.Email)
	if err != nil {
		pkghttp.WriteError(w, http.StatusInternalServerError, "internal server error")
		return
	}

	pkghttp.WriteJSON(w, http.StatusCreated, u)
}

func (h *Handler) GetByID(w http.ResponseWriter, r *http.Request) {
	var id pgtype.UUID
	if err := id.Scan(chi.URLParam(r, "id")); err != nil {
		pkghttp.WriteError(w, http.StatusBadRequest, "invalid id")
		return
	}

	u, err := h.svc.GetByID(r.Context(), id)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			pkghttp.WriteError(w, http.StatusNotFound, "user not found")
			return
		}
		pkghttp.WriteError(w, http.StatusInternalServerError, "internal server error")
		return
	}

	pkghttp.WriteJSON(w, http.StatusOK, u)
}
