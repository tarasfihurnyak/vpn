package user

import (
	"time"

	"github.com/jackc/pgx/v5/pgtype"
)

type User struct {
	ID        pgtype.UUID `json:"id"`
	Username  string      `json:"username"`
	Email     string      `json:"email"`
	PublicKey string      `json:"public_key,omitempty"`
	CreatedAt time.Time   `json:"created_at"`
	UpdatedAt time.Time   `json:"updated_at"`
}
