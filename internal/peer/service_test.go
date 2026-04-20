package peer_test

import (
	"context"
	"net/netip"
	"testing"

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
	return user.NewServiceWithMinBcryptCost(q), peer.NewService(q), ctx
}

func createTestUser(t *testing.T, svc *user.Service, ctx context.Context, username string) user.User {
	t.Helper()
	u, err := svc.Create(ctx, username, username+"@example.com", "testpassword123")
	require.NoError(t, err)
	return u
}

func TestPeerService_Create(t *testing.T) {
	tests := []struct {
		name   string
		pName  string
		pubkey string
		addr   netip.Addr
	}{
		{name: "laptop", pName: "laptop", pubkey: "pubkey-laptop", addr: netip.MustParseAddr("10.0.0.2")},
		{name: "phone", pName: "phone", pubkey: "pubkey-phone", addr: netip.MustParseAddr("10.0.0.3")},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			userSvc, peerSvc, ctx := newSvcs(t)
			u := createTestUser(t, userSvc, ctx, "peer-owner-create-"+tc.name)

			p, err := peerSvc.Create(ctx, u.ID, tc.pName, tc.pubkey, tc.addr)
			require.NoError(t, err)
			require.Equal(t, tc.pName, p.Name)
			require.Equal(t, u.ID, p.UserID)
			require.True(t, p.Enabled)
		})
	}
}

func TestPeerService_GetByID(t *testing.T) {
	userSvc, peerSvc, ctx := newSvcs(t)
	u := createTestUser(t, userSvc, ctx, "peer-owner-getbyid")

	created, err := peerSvc.Create(ctx, u.ID, "phone", "pubkey-phone", netip.MustParseAddr("10.0.0.3"))
	require.NoError(t, err)

	fetched, err := peerSvc.GetByID(ctx, created.ID)
	require.NoError(t, err)
	require.Equal(t, created, fetched)
}

func TestPeerService_GetByPublicKey(t *testing.T) {
	userSvc, peerSvc, ctx := newSvcs(t)
	u := createTestUser(t, userSvc, ctx, "peer-owner-getbypubkey")

	created, err := peerSvc.Create(ctx, u.ID, "tablet", "pubkey-tablet-unique", netip.MustParseAddr("10.0.0.4"))
	require.NoError(t, err)

	fetched, err := peerSvc.GetByPublicKey(ctx, "pubkey-tablet-unique")
	require.NoError(t, err)
	require.Equal(t, created, fetched)
}

func TestPeerService_ListByUser(t *testing.T) {
	userSvc, peerSvc, ctx := newSvcs(t)
	u := createTestUser(t, userSvc, ctx, "peer-owner-list")

	created, err := peerSvc.Create(ctx, u.ID, "device", "pubkey-device-list", netip.MustParseAddr("10.0.0.5"))
	require.NoError(t, err)

	peers, err := peerSvc.ListByUser(ctx, u.ID)
	require.NoError(t, err)
	require.Equal(t, []peer.Peer{created}, peers)
}

func TestPeerService_EnableDisable(t *testing.T) {
	userSvc, peerSvc, ctx := newSvcs(t)
	u := createTestUser(t, userSvc, ctx, "peer-owner-enabledisable")

	p, err := peerSvc.Create(ctx, u.ID, "desktop", "pubkey-desktop", netip.MustParseAddr("10.0.0.6"))
	require.NoError(t, err)
	require.True(t, p.Enabled)

	tests := []struct {
		name    string
		action  func() error
		enabled bool
	}{
		{name: "disable", action: func() error { return peerSvc.Disable(ctx, p.ID) }, enabled: false},
		{name: "enable", action: func() error { return peerSvc.Enable(ctx, p.ID) }, enabled: true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			require.NoError(t, tc.action())
			got, err := peerSvc.GetByID(ctx, p.ID)
			require.NoError(t, err)
			require.Equal(t, tc.enabled, got.Enabled)
		})
	}
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
		require.True(t, ep.Enabled)
	}
}
