package peer_test

import (
	"context"
	"log"
	"os"
	"path/filepath"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
	migrate "github.com/rubenv/sql-migrate"

	pkgdb "vpn/pkg/db"
	pkgtest "vpn/pkg/test"
)

var globalPool *pgxpool.Pool

func TestMain(m *testing.M) {
	ctx := context.Background()

	dsn, stop, err := pkgtest.StartPostgres(ctx)
	if err != nil {
		log.Fatalf("start postgres: %v", err)
	}
	defer stop()

	sqlDB, err := pkgdb.Connect(dsn)
	if err != nil {
		log.Fatalf("connect for migrations: %v", err)
	}
	migrationsDir := filepath.Join(pkgtest.MustFindRepoRoot(), pkgdb.MigrationsDir)
	if _, err := pkgdb.RunMigrations(sqlDB, migrationsDir, migrate.Up); err != nil {
		log.Fatalf("run migrations: %v", err)
	}
	_ = sqlDB.Close()

	globalPool, err = pgxpool.New(ctx, dsn)
	if err != nil {
		log.Fatalf("open pgxpool: %v", err)
	}
	defer globalPool.Close()

	os.Exit(m.Run())
}
