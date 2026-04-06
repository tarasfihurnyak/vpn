package db_test

import (
	"context"
	"path/filepath"
	"testing"

	migrate "github.com/rubenv/sql-migrate"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	pkgdb "vpn/pkg/db"
	"vpn/pkg/testhelpers"
)

func TestMigrateUpDown(t *testing.T) {
	ctx := context.Background()

	dsn, stop, err := testhelpers.StartPostgres(ctx)
	require.NoError(t, err)
	t.Cleanup(stop)

	sqlDB, err := pkgdb.Connect(dsn)
	require.NoError(t, err)
	t.Cleanup(func() { _ = sqlDB.Close() })

	migrationsDir := filepath.Join(testhelpers.MustFindRepoRoot(), pkgdb.MigrationsDir)

	n, err := pkgdb.RunMigrations(sqlDB, migrationsDir, migrate.Up)
	require.NoError(t, err)
	assert.Greater(t, n, 0, "expected at least one migration to be applied")

	// running Up again must be idempotent
	n, err = pkgdb.RunMigrations(sqlDB, migrationsDir, migrate.Up)
	require.NoError(t, err)
	assert.Equal(t, 0, n, "re-running Up should apply 0 migrations")

	n, err = pkgdb.RunMigrations(sqlDB, migrationsDir, migrate.Down)
	require.NoError(t, err)
	assert.Greater(t, n, 0, "expected at least one migration to be rolled back")
}
