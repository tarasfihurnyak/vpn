package user_test

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
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
	svc := user.NewService(sqlcdb.New(tx))
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
		{name: "ok", body: `{"username":"alice","email":"alice@example.com"}`, expectedCode: http.StatusCreated},
		{name: "invalid json", body: `{bad}`, expectedCode: http.StatusBadRequest, expectedError: true},
		{name: "missing fields", body: `{"username":"alice"}`, expectedCode: http.StatusBadRequest, expectedError: true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			req, rec := testutil.NewJSONRequest(http.MethodPost, "/users", tc.body)

			d.h.Create(rec, req)

			assert.Equal(t, tc.expectedCode, rec.Code)
			assert.Equal(t, "application/json", rec.Header().Get("Content-Type"))
			if tc.expectedError {
				return
			}

			var got user.User
			require.NoError(t, json.NewDecoder(rec.Body).Decode(&got))
			assert.Equal(t, "alice", got.Username)
			assert.Equal(t, "alice@example.com", got.Email)
			assert.True(t, got.ID.Valid)
		})
	}
}

func TestUserHandler_GetByID(t *testing.T) {
	d := newDeps(t)

	// preconditions
	u, err := d.svc.Create(d.ctx, "bob", "bob@example.com")
	require.NoError(t, err)
	existingID := uuid.UUID(u.ID.Bytes).String()

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

			assert.Equal(t, tc.expectedCode, rec.Code)
			assert.Equal(t, "application/json", rec.Header().Get("Content-Type"))
			if tc.expectedError {
				return
			}

			var got user.User
			require.NoError(t, json.NewDecoder(rec.Body).Decode(&got))
			assert.Equal(t, u.ID, got.ID)
			assert.Equal(t, "bob", got.Username)
		})
	}
}
