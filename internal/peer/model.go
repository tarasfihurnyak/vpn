package peer

import (
	"errors"
	"net/netip"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
)

var ErrNotFound = errors.New("peer not found")

type Peer struct {
	ID        pgtype.UUID `json:"id"`
	UserID    pgtype.UUID `json:"user_id"`
	Name      string      `json:"name"`
	PublicKey string      `json:"public_key"`
	IPAddress netip.Addr  `json:"ip_address"`
	Enabled   bool        `json:"enabled"`
	CreatedAt time.Time   `json:"created_at"`
	UpdatedAt time.Time   `json:"updated_at"`
}
