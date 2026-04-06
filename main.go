package main

import (
	"os"
	"os/signal"
	"syscall"

	"vpn/pkg/config"
	"vpn/pkg/db"
	"vpn/pkg/logger"

	"github.com/rs/zerolog/log"
)

func main() {
	logger.Setup()

	cfg, err := config.Load(".env")
	if err != nil {
		log.Fatal().Err(err).Msg("load config")
	}

	log.Info().Msg("starting vpn server")

	database, err := db.Connect(cfg.DB.DSN())
	if err != nil {
		log.Fatal().Err(err).Msg("connect db")
	}
	defer func() {
		if err := database.Close(); err != nil {
			log.Error().Err(err).Msg("close db")
		}
	}()

	log.Info().Msg("connected to database")

	n, err := db.MigrateUp(database)
	if err != nil {
		log.Fatal().Err(err).Msg("run migrations")
	}
	log.Info().Int("count", n).Msg("migrations applied")

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	sig := <-quit
	log.Info().Str("signal", sig.String()).Msg("shutting down")
}
