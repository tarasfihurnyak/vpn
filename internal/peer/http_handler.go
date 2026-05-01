package peer

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/netip"

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
	UserID    string `json:"user_id"`
	Name      string `json:"name"`
	PublicKey string `json:"public_key"`
	IPAddress string `json:"ip_address"`
}

// @Summary      Create peer
// @Description  Register a new WireGuard peer for a user.
// @Tags         peers
// @Accept       json
// @Produce      json
// @Param        body  body      createRequest  true  "Peer details"
// @Success      201   {object}  Peer
// @Failure      400   {object}  pkghttp.ErrorResponse
// @Failure      500   {object}  pkghttp.ErrorResponse
// @Security     BearerAuth
// @Router       /peers [post]
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	var req createRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		var maxErr *http.MaxBytesError
		if errors.As(err, &maxErr) {
			log.Error().Err(err).Msg("create peer: request body too large")
			pkghttp.WriteError(w, http.StatusRequestEntityTooLarge, "request body too large")
			return
		}
		log.Error().Err(err).Msg("create peer: invalid request body")
		pkghttp.WriteError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.UserID == "" || req.Name == "" || req.PublicKey == "" || req.IPAddress == "" {
		log.Error().Msg("create peer: missing required fields")
		pkghttp.WriteError(w, http.StatusBadRequest, "user_id, name, public_key, and ip_address are required")
		return
	}

	userID, err := uuid.Parse(req.UserID)
	if err != nil {
		log.Error().Err(err).Str("user_id", req.UserID).Msg("create peer: invalid user_id")
		pkghttp.WriteError(w, http.StatusBadRequest, "invalid user_id")
		return
	}

	ipAddr, err := netip.ParseAddr(req.IPAddress)
	if err != nil {
		log.Error().Err(err).Str("ip_address", req.IPAddress).Msg("create peer: invalid ip_address")
		pkghttp.WriteError(w, http.StatusBadRequest, "invalid ip_address")
		return
	}

	p, err := h.svc.Create(r.Context(), userID, req.Name, req.PublicKey, ipAddr)
	if err != nil {
		log.Error().Err(err).Str("user_id", req.UserID).Str("name", req.Name).Msg("create peer failed")
		pkghttp.WriteError(w, http.StatusInternalServerError, "internal server error")
		return
	}

	pkghttp.WriteJSON(w, http.StatusCreated, p)
}

// @Summary      Get peer
// @Description  Retrieve a WireGuard peer by ID.
// @Tags         peers
// @Produce      json
// @Param        id   path      string  true  "Peer UUID"
// @Success      200  {object}  Peer
// @Failure      400  {object}  pkghttp.ErrorResponse
// @Failure      404  {object}  pkghttp.ErrorResponse
// @Failure      500  {object}  pkghttp.ErrorResponse
// @Security     BearerAuth
// @Router       /peers/{id} [get]
func (h *Handler) GetByID(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		log.Error().Err(err).Str("id", chi.URLParam(r, "id")).Msg("get peer: invalid id")
		pkghttp.WriteError(w, http.StatusBadRequest, "invalid id")
		return
	}

	p, err := h.svc.GetByID(r.Context(), id)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			log.Error().Str("id", id.String()).Msg("get peer: not found")
			pkghttp.WriteError(w, http.StatusNotFound, "peer not found")
			return
		}
		log.Error().Err(err).Str("id", id.String()).Msg("get peer: internal error")
		pkghttp.WriteError(w, http.StatusInternalServerError, "internal server error")
		return
	}

	pkghttp.WriteJSON(w, http.StatusOK, p)
}
