package testutil

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"vpn/internal/user"
)

const Password = "testpassword123"

func CreateUser(t *testing.T, svc *user.Service, ctx context.Context, name string) user.User {
	t.Helper()
	u, err := svc.Create(ctx, name, name+"@example.com", Password)
	require.NoError(t, err)
	return u
}
