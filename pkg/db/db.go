package db

import (
	"database/sql"
	"fmt"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
	migrate "github.com/rubenv/sql-migrate"
)

const MigrationsDir = "db/migrations"

func Connect(dsn string) (*sql.DB, error) {
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		return nil, fmt.Errorf("open: %w", err)
	}

	db.SetMaxOpenConns(10)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)
	db.SetConnMaxIdleTime(1 * time.Minute)

	if err := db.Ping(); err != nil {
		db.Close()
		return nil, fmt.Errorf("ping: %w", err)
	}

	return db, nil
}

func RunMigrations(database *sql.DB, dir string, direction migrate.MigrationDirection) (int, error) {
	n, err := migrate.Exec(database, "postgres", &migrate.FileMigrationSource{Dir: dir}, direction)
	if err != nil {
		return 0, fmt.Errorf("migrate: %w", err)
	}
	return n, nil
}

func MigrateUp(database *sql.DB) (int, error) {
	return RunMigrations(database, MigrationsDir, migrate.Up)
}

func MigrateDown(database *sql.DB) (int, error) {
	return RunMigrations(database, MigrationsDir, migrate.Down)
}
