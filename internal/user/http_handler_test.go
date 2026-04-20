package user_test

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	sqlcdb "vpn/internal/db/sqlc"
	"vpn/internal/testutil"
	"vpn/internal/user"
)

type deps struct {
	ctx context.Context
	h   *user.Handler
	svc *user.Service
}

func newDeps(t *testing.T) deps {
	t.Helper()
	ctx := context.Background()
	tx, err := globalPool.Begin(ctx)
	require.NoError(t, err)
	t.Cleanup(func() { _ = tx.Rollback(ctx) })
	svc := user.NewServiceWithMinBcryptCost(sqlcdb.New(tx))
	return deps{ctx: ctx, h: user.NewHandler(svc), svc: svc}
}

func TestUserHandler_Create(t *testing.T) {
	d := newDeps(t)

	tests := []struct {
		name          string
		body          string
		expectedCode  int
		expectedError bool
	}{
		{name: "ok", body: `{"username":"alice","email":"alice@example.com","password":"password123"}`, expectedCode: http.StatusCreated},
		{name: "invalid json", body: `{bad}`, expectedCode: http.StatusBadRequest, expectedError: true},
		{name: "missing fields", body: `{"username":"alice"}`, expectedCode: http.StatusBadRequest, expectedError: true},
		{name: "short password", body: `{"username":"alice","email":"alice@example.com","password":"short"}`, expectedCode: http.StatusBadRequest, expectedError: true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			req, rec := testutil.NewJSONRequest(http.MethodPost, "/users", tc.body)

			d.h.Create(rec, req)

			require.Equal(t, tc.expectedCode, rec.Code)
			require.Equal(t, "application/json", rec.Header().Get("Content-Type"))
			if tc.expectedError {
				return
			}

			var got user.User
			require.NoError(t, json.NewDecoder(rec.Body).Decode(&got))
			require.Equal(t, "alice", got.Username)
			require.Equal(t, "alice@example.com", got.Email)
			require.NotEqual(t, uuid.Nil, got.ID)
		})
	}
}

func TestUserHandler_GetByID(t *testing.T) {
	d := newDeps(t)

	// preconditions
	u, err := d.svc.Create(d.ctx, "bob", "bob@example.com", "password123")
	require.NoError(t, err)
	existingID := u.ID.String()

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
			req, rec := testutil.NewJSONRequest(http.MethodGet, "/users/"+tc.id, "")
			req = testutil.WithURLParam(req, "id", tc.id)

			d.h.GetByID(rec, req)

			require.Equal(t, tc.expectedCode, rec.Code)
			require.Equal(t, "application/json", rec.Header().Get("Content-Type"))
			if tc.expectedError {
				return
			}

			var got user.User
			require.NoError(t, json.NewDecoder(rec.Body).Decode(&got))
			require.Equal(t, u.ID, got.ID)
			require.Equal(t, "bob", got.Username)
		})
	}
}
