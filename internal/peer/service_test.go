package peer_test

import (
	"context"
	"net/netip"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	sqlcdb "vpn/internal/db/sqlc"
	"vpn/internal/peer"
	"vpn/internal/user"
)

func newSvcs(t *testing.T) (*user.Service, *peer.Service, context.Context) {
	t.Helper()
	ctx := context.Background()
	tx, err := globalPool.Begin(ctx)
	require.NoError(t, err)
	t.Cleanup(func() { _ = tx.Rollback(ctx) })
	q := sqlcdb.New(tx)
	return user.NewService(q), peer.NewService(q), ctx
}

func createTestUser(t *testing.T, svc *user.Service, ctx context.Context, username string) user.User {
	t.Helper()
	u, err := svc.Create(ctx, username, username+"@example.com")
	require.NoError(t, err)
	return u
}

func TestPeerService_Create(t *testing.T) {
	userSvc, peerSvc, ctx := newSvcs(t)
	u := createTestUser(t, userSvc, ctx, "peer-owner-create")

	p, err := peerSvc.Create(ctx, u.ID, "laptop", "pubkey-laptop", netip.MustParseAddr("10.0.0.2"))
	require.NoError(t, err)
	assert.Equal(t, "laptop", p.Name)
	assert.Equal(t, u.ID, p.UserID)
	assert.True(t, p.Enabled)
}

func TestPeerService_GetByID(t *testing.T) {
	userSvc, peerSvc, ctx := newSvcs(t)
	u := createTestUser(t, userSvc, ctx, "peer-owner-getbyid")

	created, err := peerSvc.Create(ctx, u.ID, "phone", "pubkey-phone", netip.MustParseAddr("10.0.0.3"))
	require.NoError(t, err)

	fetched, err := peerSvc.GetByID(ctx, created.ID)
	require.NoError(t, err)
	assert.Equal(t, created.ID, fetched.ID)
	assert.Equal(t, "phone", fetched.Name)
}

func TestPeerService_GetByPublicKey(t *testing.T) {
	userSvc, peerSvc, ctx := newSvcs(t)
	u := createTestUser(t, userSvc, ctx, "peer-owner-getbypubkey")

	_, err := peerSvc.Create(ctx, u.ID, "tablet", "pubkey-tablet-unique", netip.MustParseAddr("10.0.0.4"))
	require.NoError(t, err)

	p, err := peerSvc.GetByPublicKey(ctx, "pubkey-tablet-unique")
	require.NoError(t, err)
	assert.Equal(t, "tablet", p.Name)
}

func TestPeerService_ListByUser(t *testing.T) {
	userSvc, peerSvc, ctx := newSvcs(t)
	u := createTestUser(t, userSvc, ctx, "peer-owner-list")

	_, err := peerSvc.Create(ctx, u.ID, "device", "pubkey-device-list", netip.MustParseAddr("10.0.0.5"))
	require.NoError(t, err)

	peers, err := peerSvc.ListByUser(ctx, u.ID)
	require.NoError(t, err)
	assert.Len(t, peers, 1)
	assert.Equal(t, "device", peers[0].Name)
}

func TestPeerService_EnableDisable(t *testing.T) {
	userSvc, peerSvc, ctx := newSvcs(t)
	u := createTestUser(t, userSvc, ctx, "peer-owner-enabledisable")

	p, err := peerSvc.Create(ctx, u.ID, "desktop", "pubkey-desktop", netip.MustParseAddr("10.0.0.6"))
	require.NoError(t, err)
	assert.True(t, p.Enabled)

	require.NoError(t, peerSvc.Disable(ctx, p.ID))
	disabled, err := peerSvc.GetByID(ctx, p.ID)
	require.NoError(t, err)
	assert.False(t, disabled.Enabled)

	require.NoError(t, peerSvc.Enable(ctx, p.ID))
	enabled, err := peerSvc.GetByID(ctx, p.ID)
	require.NoError(t, err)
	assert.True(t, enabled.Enabled)
}

func TestPeerService_ListEnabled(t *testing.T) {
	userSvc, peerSvc, ctx := newSvcs(t)
	u := createTestUser(t, userSvc, ctx, "peer-owner-listenabled")

	p, err := peerSvc.Create(ctx, u.ID, "router", "pubkey-router", netip.MustParseAddr("10.0.0.7"))
	require.NoError(t, err)

	require.NoError(t, peerSvc.Disable(ctx, p.ID))

	enabled, err := peerSvc.ListEnabled(ctx)
	require.NoError(t, err)
	for _, ep := range enabled {
		assert.True(t, ep.Enabled)
	}
}
