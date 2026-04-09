package user_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
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
	return user.NewService(sqlcdb.New(tx)), ctx
}

func TestUserService_Create(t *testing.T) {
	svc, ctx := newSvc(t)

	u, err := svc.Create(ctx, "alice", "alice@example.com")
	require.NoError(t, err)
	assert.Equal(t, "alice", u.Username)
	assert.Equal(t, "alice@example.com", u.Email)
	assert.True(t, u.ID.Valid)
}

func TestUserService_GetByID(t *testing.T) {
	svc, ctx := newSvc(t)

	created, err := svc.Create(ctx, "bob", "bob@example.com")
	require.NoError(t, err)

	fetched, err := svc.GetByID(ctx, created.ID)
	require.NoError(t, err)
	assert.Equal(t, created.ID, fetched.ID)
	assert.Equal(t, "bob", fetched.Username)
}

func TestUserService_GetByEmail(t *testing.T) {
	svc, ctx := newSvc(t)

	_, err := svc.Create(ctx, "carol", "carol@example.com")
	require.NoError(t, err)

	u, err := svc.GetByEmail(ctx, "carol@example.com")
	require.NoError(t, err)
	assert.Equal(t, "carol", u.Username)
}

func TestUserService_GetByUsername(t *testing.T) {
	svc, ctx := newSvc(t)

	_, err := svc.Create(ctx, "dave", "dave@example.com")
	require.NoError(t, err)

	u, err := svc.GetByUsername(ctx, "dave")
	require.NoError(t, err)
	assert.Equal(t, "dave@example.com", u.Email)
}

func TestUserService_List(t *testing.T) {
	svc, ctx := newSvc(t)

	for _, name := range []string{"u1", "u2", "u3"} {
		_, err := svc.Create(ctx, name, name+"@example.com")
		require.NoError(t, err)
	}

	users, err := svc.List(ctx)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(users), 3)
}

func TestUserService_UpdatePublicKey(t *testing.T) {
	svc, ctx := newSvc(t)

	created, err := svc.Create(ctx, "eve", "eve@example.com")
	require.NoError(t, err)

	updated, err := svc.UpdatePublicKey(ctx, created.ID, "pubkey-abc")
	require.NoError(t, err)
	assert.Equal(t, "pubkey-abc", updated.PublicKey)
}

func TestUserService_Delete(t *testing.T) {
	svc, ctx := newSvc(t)

	created, err := svc.Create(ctx, "frank", "frank@example.com")
	require.NoError(t, err)

	err = svc.Delete(ctx, created.ID)
	require.NoError(t, err)

	_, err = svc.GetByID(ctx, created.ID)
	assert.Error(t, err, "deleted user should not be found")
}
