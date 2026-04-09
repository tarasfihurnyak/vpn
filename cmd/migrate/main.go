package main

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"vpn/pkg/config"
	"vpn/pkg/db"
	"vpn/pkg/logger"

	"github.com/rs/zerolog/log"
	migrate "github.com/rubenv/sql-migrate"
)

func init() {
	logger.Setup()
}

func main() {
	if len(os.Args) < 2 {
		log.Fatal().Msg("usage: migrate <up|down|new <name>>")
	}

	switch os.Args[1] {
	case "new":
		if len(os.Args) < 3 {
			log.Fatal().Msg("usage: migrate new <name>")
		}
		createMigration(os.Args[2])
	case "up", "down":
		cfg, err := config.Load(".env")
		if err != nil {
			log.Fatal().Err(err).Msg("load config")
		}
		direction, err := parseDirection(os.Args[1])
		if err != nil {
			log.Fatal().Err(err).Msg("invalid direction")
		}
		runMigrations(cfg.DB.DSN(), direction)
	default:
		log.Fatal().Str("command", os.Args[1]).Msg("unknown command")
	}
}

func createMigration(name string) {
	timestamp := time.Now().Format("20060102150405")
	filename := filepath.Join(db.MigrationsDir, fmt.Sprintf("%s_%s.sql", timestamp, name))

	content := "-- +migrate Up\n\n\n-- +migrate Down\n"
	if err := os.WriteFile(filename, []byte(content), 0644); err != nil {
		log.Fatal().Err(err).Str("file", filename).Msg("create migration")
	}
	log.Info().Str("file", filename).Msg("migration file created")
}

func runMigrations(dsn string, direction migrate.MigrationDirection) {
	database, err := db.Connect(dsn)
	if err != nil {
		log.Fatal().Err(err).Msg("connect db")
	}
	defer func() {
		if err := database.Close(); err != nil {
			log.Error().Err(err).Msg("close db")
		}
	}()

	n, err := db.RunMigrations(database, db.MigrationsDir, direction)
	if err != nil {
		log.Fatal().Err(err).Msg("migrate")
	}

	log.Info().Int("count", n).Msg("migrations applied")
}

func parseDirection(s string) (migrate.MigrationDirection, error) {
	switch s {
	case "up":
		return migrate.Up, nil
	case "down":
		return migrate.Down, nil
	default:
		return 0, fmt.Errorf("unknown direction %q: must be up or down", s)
	}
}
