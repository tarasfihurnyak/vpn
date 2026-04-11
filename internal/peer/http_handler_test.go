package peer_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/netip"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	sqlcdb "vpn/internal/db/sqlc"
	"vpn/internal/peer"
	"vpn/internal/testutil"
	"vpn/internal/user"
)

type deps struct {
	ctx   context.Context
	h     *peer.Handler
	peers *peer.Service
	users *user.Service
}

func newDeps(t *testing.T) deps {
	t.Helper()
	ctx := context.Background()
	tx, err := globalPool.Begin(ctx)
	require.NoError(t, err)
	t.Cleanup(func() { _ = tx.Rollback(ctx) })
	q := sqlcdb.New(tx)
	peerSvc := peer.NewService(q)
	userSvc := user.NewService(q)
	return deps{ctx: ctx, h: peer.NewHandler(peerSvc), peers: peerSvc, users: userSvc}
}

func TestPeerHandler_Create(t *testing.T) {
	d := newDeps(t)

	// preconditions
	u := testutil.CreateUser(t, d.users, d.ctx, "create-owner")
	validBody := fmt.Sprintf(`{"user_id":"%s","name":"laptop","public_key":"pubkey-abc","ip_address":"10.0.0.2"}`,
		uuid.UUID(u.ID.Bytes).String())
	badIPBody := fmt.Sprintf(`{"user_id":"%s","name":"laptop","public_key":"k","ip_address":"not-an-ip"}`,
		uuid.UUID(u.ID.Bytes).String())

	tests := []struct {
		name          string
		body          string
		expectedCode  int
		expectedError bool
	}{
		{name: "ok", body: validBody, expectedCode: http.StatusCreated},
		{name: "invalid json", body: `{bad}`, expectedCode: http.StatusBadRequest, expectedError: true},
		{name: "missing fields", body: `{"name":"laptop"}`, expectedCode: http.StatusBadRequest, expectedError: true},
		{name: "invalid user_id", body: `{"user_id":"not-a-uuid","name":"laptop","public_key":"k","ip_address":"10.0.0.1"}`, expectedCode: http.StatusBadRequest, expectedError: true},
		{name: "invalid ip_address", body: badIPBody, expectedCode: http.StatusBadRequest, expectedError: true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			req, rec := testutil.NewJSONRequest(http.MethodPost, "/peers", tc.body)

			d.h.Create(rec, req)

			assert.Equal(t, tc.expectedCode, rec.Code)
			assert.Equal(t, "application/json", rec.Header().Get("Content-Type"))
			if tc.expectedError {
				return
			}

			var got peer.Peer
			require.NoError(t, json.NewDecoder(rec.Body).Decode(&got))
			assert.Equal(t, "laptop", got.Name)
			assert.Equal(t, u.ID, got.UserID)
			assert.Equal(t, "10.0.0.2", got.IPAddress.String())
			assert.True(t, got.Enabled)
		})
	}
}

func TestPeerHandler_GetByID(t *testing.T) {
	d := newDeps(t)

	// preconditions
	u := testutil.CreateUser(t, d.users, d.ctx, "getbyid-owner")
	p, err := d.peers.Create(d.ctx, u.ID, "phone", "pubkey-phone", netip.MustParseAddr("10.0.0.3"))
	require.NoError(t, err)
	existingID := uuid.UUID(p.ID.Bytes).String()

	tests := []struct {
		name          string
		id            string
		expectedCode  int
		expectedError bool
	}{
		{name: "ok", id: existingID, expectedCode: http.StatusOK},
		{name: "invalid id", id: "not-a-uuid", expectedCode: http.StatusBadRequest, expectedError: true},
		{name: "not found", id: uuid.New().String(), expectedCode: http.StatusNotFound, expectedError: true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			req, rec := testutil.NewJSONRequest(http.MethodGet, "/peers/"+tc.id, "")
			req = testutil.WithURLParam(req, "id", tc.id)

			d.h.GetByID(rec, req)

			assert.Equal(t, tc.expectedCode, rec.Code)
			assert.Equal(t, "application/json", rec.Header().Get("Content-Type"))
			if tc.expectedError {
				return
			}

			var got peer.Peer
			require.NoError(t, json.NewDecoder(rec.Body).Decode(&got))
			assert.Equal(t, p.ID, got.ID)
			assert.Equal(t, "phone", got.Name)
		})
	}
}
