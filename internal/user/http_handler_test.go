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
		bodyLimit     int64
		expectedCode  int
		expectedError bool
	}{
		{name: "ok", body: `{"username":"alice","email":"alice@example.com","password":"password123"}`, expectedCode: http.StatusCreated},
		{name: "invalid json", body: `{bad}`, expectedCode: http.StatusBadRequest, expectedError: true},
		{name: "missing fields", body: `{"username":"alice"}`, expectedCode: http.StatusBadRequest, expectedError: true},
		{name: "short password", body: `{"username":"alice","email":"alice@example.com","password":"short"}`, expectedCode: http.StatusBadRequest, expectedError: true},
		{name: "body too large", body: `{"username":"alice","email":"alice@example.com","password":"password123"}`, bodyLimit: 1, expectedCode: http.StatusRequestEntityTooLarge, expectedError: true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			req, rec := testutil.NewJSONRequest(http.MethodPost, "/users", tc.body)
			if tc.bodyLimit > 0 {
				req.Body = http.MaxBytesReader(rec, req.Body, tc.bodyLimit)
			}

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
	u := testutil.CreateUser(t, d.svc, d.ctx, "bob")

	tests := []struct {
		name          string
		id            uuid.UUID
		expectedCode  int
		expectedError bool
	}{
		{name: "ok", id: u.ID, expectedCode: http.StatusOK},
		{name: "invalid id", id: uuid.Nil, expectedCode: http.StatusBadRequest, expectedError: true},
		{name: "not found", id: uuid.New(), expectedCode: http.StatusNotFound, expectedError: true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			req, rec := testutil.NewJSONRequest(http.MethodGet, "/users/"+tc.id.String(), "")
			req = testutil.WithURLParam(req, "id", tc.id.String())

			d.h.GetByID(rec, req)

			require.Equal(t, tc.expectedCode, rec.Code)
			require.Equal(t, "application/json", rec.Header().Get("Content-Type"))
			if tc.expectedError {
				return
			}

			var got user.User
			require.NoError(t, json.NewDecoder(rec.Body).Decode(&got))
			require.Equal(t, tc.id, got.ID)
			require.Equal(t, "bob", got.Username)
		})
	}
}

func TestUserHandler_List(t *testing.T) {
	d := newDeps(t)

	u1 := testutil.CreateUser(t, d.svc, d.ctx, "alice")
	u2 := testutil.CreateUser(t, d.svc, d.ctx, "bob")

	req, rec := testutil.NewJSONRequest(http.MethodGet, "/users", "")
	d.h.List(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	require.Equal(t, "application/json", rec.Header().Get("Content-Type"))

	var got []user.User
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&got))
	require.Len(t, got, 2)
	require.ElementsMatch(t, []user.User{u1, u2}, got)
}
