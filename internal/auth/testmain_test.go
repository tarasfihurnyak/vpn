package auth_test

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"log"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	migrate "github.com/rubenv/sql-migrate"

	"vpn/internal/auth"
	sqlcdb "vpn/internal/db/sqlc"
	"vpn/internal/user"
	pkgdb "vpn/pkg/db"
	pkgtest "vpn/pkg/test"
)

var (
	globalPool *pgxpool.Pool
	testKey    *ecdsa.PrivateKey
)

const (
	testAccessTTL  = 15 * time.Minute
	testRefreshTTL = 7 * 24 * time.Hour
)

func TestMain(m *testing.M) {
	ctx := context.Background()

	var err error
	testKey, err = ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		log.Fatalf("generate test EC key: %v", err)
	}

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

// newDeps creates a transactional test scope and returns auth + user services.
func newDeps(t *testing.T) (*auth.Service, *user.Service, context.Context) {
	t.Helper()
	ctx := context.Background()
	tx, err := globalPool.Begin(ctx)
	if err != nil {
		t.Fatalf("begin tx: %v", err)
	}
	t.Cleanup(func() { _ = tx.Rollback(ctx) })

	q := sqlcdb.New(tx)
	userSvc := user.NewServiceWithMinBcryptCost(q)
	authSvc := auth.NewService(userSvc, q, testKey, testAccessTTL, testRefreshTTL)
	return authSvc, userSvc, ctx
}
