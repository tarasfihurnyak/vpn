package peer

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/netip"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	pkghttp "vpn/pkg/http"
)

type Handler struct {
	svc *Service
}

func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

type createRequest struct {
	UserID    string `json:"user_id"`
	Name      string `json:"name"`
	PublicKey string `json:"public_key"`
	IPAddress string `json:"ip_address"`
}

func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	var req createRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		pkghttp.WriteError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.UserID == "" || req.Name == "" || req.PublicKey == "" || req.IPAddress == "" {
		pkghttp.WriteError(w, http.StatusBadRequest, "user_id, name, public_key, and ip_address are required")
		return
	}

	userID, err := uuid.Parse(req.UserID)
	if err != nil {
		pkghttp.WriteError(w, http.StatusBadRequest, "invalid user_id")
		return
	}

	ipAddr, err := netip.ParseAddr(req.IPAddress)
	if err != nil {
		pkghttp.WriteError(w, http.StatusBadRequest, "invalid ip_address")
		return
	}

	p, err := h.svc.Create(r.Context(), userID, req.Name, req.PublicKey, ipAddr)
	if err != nil {
		pkghttp.WriteError(w, http.StatusInternalServerError, "internal server error")
		return
	}

	pkghttp.WriteJSON(w, http.StatusCreated, p)
}

func (h *Handler) GetByID(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		pkghttp.WriteError(w, http.StatusBadRequest, "invalid id")
		return
	}

	p, err := h.svc.GetByID(r.Context(), id)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			pkghttp.WriteError(w, http.StatusNotFound, "peer not found")
			return
		}
		pkghttp.WriteError(w, http.StatusInternalServerError, "internal server error")
		return
	}

	pkghttp.WriteJSON(w, http.StatusOK, p)
}
