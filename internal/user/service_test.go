package user_test

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	sqlcdb "vpn/internal/db/sqlc"
	"vpn/internal/user"
)

func newSvc(t *testing.T) (*user.Service, context.Context) {
	t.Helper()
	ctx := context.Background()
	tx, err := globalPool.Begin(ctx)
	require.NoError(t, err)
	t.Cleanup(func() { _ = tx.Rollback(ctx) })
	return user.NewServiceWithMinBcryptCost(sqlcdb.New(tx)), ctx
}

func TestUserService_Create(t *testing.T) {
	tests := []struct {
		name     string
		username string
		email    string
	}{
		{name: "basic user", username: "alice", email: "alice@example.com"},
		{name: "another user", username: "bob", email: "bob@example.com"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			svc, ctx := newSvc(t)

			u, err := svc.Create(ctx, tc.username, tc.email, "password123")
			require.NoError(t, err)
			require.Equal(t, tc.username, u.Username)
			require.Equal(t, tc.email, u.Email)
			require.NotEqual(t, uuid.Nil, u.ID)
		})
	}
}

func TestUserService_GetByID(t *testing.T) {
	svc, ctx := newSvc(t)

	created, err := svc.Create(ctx, "bob", "bob@example.com", "password123")
	require.NoError(t, err)

	fetched, err := svc.GetByID(ctx, created.ID)
	require.NoError(t, err)
	require.Equal(t, created, fetched)
}

func TestUserService_GetByEmail(t *testing.T) {
	svc, ctx := newSvc(t)

	created, err := svc.Create(ctx, "carol", "carol@example.com", "password123")
	require.NoError(t, err)

	fetched, err := svc.GetByEmail(ctx, "carol@example.com")
	require.NoError(t, err)
	require.Equal(t, created, fetched)
}

func TestUserService_GetByUsername(t *testing.T) {
	svc, ctx := newSvc(t)

	created, err := svc.Create(ctx, "dave", "dave@example.com", "password123")
	require.NoError(t, err)

	fetched, err := svc.GetByUsername(ctx, "dave")
	require.NoError(t, err)
	require.Equal(t, created, fetched)
}

func TestUserService_List(t *testing.T) {
	svc, ctx := newSvc(t)

	for _, name := range []string{"u1", "u2", "u3"} {
		_, err := svc.Create(ctx, name, name+"@example.com", "password123")
		require.NoError(t, err)
	}

	users, err := svc.List(ctx)
	require.NoError(t, err)
	require.GreaterOrEqual(t, len(users), 3)
}

func TestUserService_UpdatePublicKey(t *testing.T) {
	tests := []struct {
		name      string
		publicKey string
	}{
		{name: "set key", publicKey: "pubkey-abc"},
		{name: "update key", publicKey: "pubkey-xyz"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			svc, ctx := newSvc(t)

			created, err := svc.Create(ctx, "eve", "eve@example.com", "password123")
			require.NoError(t, err)

			updated, err := svc.UpdatePublicKey(ctx, created.ID, tc.publicKey)
			require.NoError(t, err)
			require.Equal(t, tc.publicKey, updated.PublicKey)
		})
	}
}

func TestUserService_Delete(t *testing.T) {
	svc, ctx := newSvc(t)

	created, err := svc.Create(ctx, "frank", "frank@example.com", "password123")
	require.NoError(t, err)

	err = svc.Delete(ctx, created.ID)
	require.NoError(t, err)

	_, err = svc.GetByID(ctx, created.ID)
	require.Error(t, err, "deleted user should not be found")
}
