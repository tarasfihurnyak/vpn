package auth_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"vpn/internal/auth"
)

func TestAuthService_Login(t *testing.T) {
	svc, userSvc, ctx := newDeps(t)

	_, err := userSvc.Create(ctx, "loginuser", "loginuser@example.com", "hunter2password")
	require.NoError(t, err)

	tests := []struct {
		name          string
		login         string
		password      string
		expectedError error
	}{
		{name: "by username", login: "loginuser", password: "hunter2password"},
		{name: "by email", login: "loginuser@example.com", password: "hunter2password"},
		{name: "wrong password", login: "loginuser", password: "wrongpassword", expectedError: auth.ErrInvalidCredentials},
		{name: "user not found", login: "ghost", password: "hunter2password", expectedError: auth.ErrInvalidCredentials},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			pair, rawRefresh, err := svc.Login(ctx, tc.login, tc.password)
			if tc.expectedError != nil {
				require.ErrorIs(t, err, tc.expectedError)
				return
			}
			require.NoError(t, err)
			require.NotEmpty(t, pair.AccessToken)
			require.NotEmpty(t, rawRefresh)
		})
	}
}

func TestAuthService_Refresh(t *testing.T) {
	svc, userSvc, ctx := newDeps(t)

	_, err := userSvc.Create(ctx, "refreshuser", "refreshuser@example.com", "password123")
	require.NoError(t, err)

	t.Run("success", func(t *testing.T) {
		_, rawRefresh, err := svc.Login(ctx, "refreshuser", "password123")
		require.NoError(t, err)

		pair2, rawRefresh2, err := svc.Refresh(ctx, rawRefresh)
		require.NoError(t, err)
		require.NotEmpty(t, pair2.AccessToken)
		require.NotEmpty(t, rawRefresh2)
		require.NotEqual(t, rawRefresh, rawRefresh2, "rotated token should differ")
	})

	t.Run("old token revoked after rotation", func(t *testing.T) {
		_, rawRefresh, err := svc.Login(ctx, "refreshuser", "password123")
		require.NoError(t, err)

		_, _, err = svc.Refresh(ctx, rawRefresh)
		require.NoError(t, err)

		_, _, err = svc.Refresh(ctx, rawRefresh)
		require.Error(t, err)
	})

	t.Run("invalid token", func(t *testing.T) {
		_, _, err := svc.Refresh(ctx, "totally-bogus-token")
		require.ErrorIs(t, err, auth.ErrInvalidToken)
	})
}

func TestAuthService_Logout(t *testing.T) {
	svc, userSvc, ctx := newDeps(t)

	_, err := userSvc.Create(ctx, "logoutuser", "logoutuser@example.com", "password123")
	require.NoError(t, err)

	_, rawRefresh, err := svc.Login(ctx, "logoutuser", "password123")
	require.NoError(t, err)

	require.NoError(t, svc.Logout(ctx, rawRefresh))

	_, _, err = svc.Refresh(ctx, rawRefresh)
	require.Error(t, err)
}

func TestAuthService_ValidateAccessToken(t *testing.T) {
	svc, userSvc, ctx := newDeps(t)

	u, err := userSvc.Create(ctx, "tokenuser", "tokenuser@example.com", "password123")
	require.NoError(t, err)

	pair, _, err := svc.Login(ctx, "tokenuser", "password123")
	require.NoError(t, err)

	tests := []struct {
		name    string
		token   string
		wantErr error
	}{
		{name: "valid token", token: pair.AccessToken},
		{name: "invalid token", token: "not.a.jwt", wantErr: auth.ErrInvalidToken},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			userID, err := svc.ValidateAccessToken(tc.token)
			if tc.wantErr != nil {
				require.ErrorIs(t, err, tc.wantErr)
				return
			}
			require.NoError(t, err)
			require.Equal(t, u.ID, userID)
		})
	}
}
