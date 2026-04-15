package testhelpers

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
)

// MustFindRepoRoot walks up from the current working directory until it finds
// go.mod. Panics if not found. Safe to call from both TestMain and individual
// test helpers.
func MustFindRepoRoot() string {
	dir, err := os.Getwd()
	if err != nil {
		panic(fmt.Sprintf("testhelpers: getwd: %v", err))
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			panic("testhelpers: go.mod not found in any parent directory")
		}
		dir = parent
	}
}

// StartPostgres starts a postgres:16-alpine testcontainer and returns the DSN
// and a stop function that terminates the container. The caller must call stop
// when done (e.g. defer stop() or t.Cleanup(stop)).
func StartPostgres(ctx context.Context) (dsn string, stop func(), err error) {
	ctr, err := postgres.Run(ctx,
		"postgres:16-alpine",
		postgres.WithDatabase("vpn_test"),
		postgres.WithUsername("postgres"),
		postgres.WithPassword("postgres"),
		testcontainers.WithWaitStrategy(
			wait.ForAll(
				wait.ForListeningPort("5432/tcp").
					WithStartupTimeout(30*time.Second),
				wait.ForLog("database system is ready to accept connections").
					WithStartupTimeout(30*time.Second).
					WithPollInterval(200*time.Millisecond),
			),
		),
	)
	if err != nil {
		return "", nil, fmt.Errorf("start postgres container: %w", err)
	}

	dsn, err = ctr.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		_ = ctr.Terminate(ctx)
		return "", nil, fmt.Errorf("get connection string: %w", err)
	}

	stop = func() { _ = ctr.Terminate(ctx) }
	return dsn, stop, nil
}
