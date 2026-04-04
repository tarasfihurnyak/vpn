package main

import (
	"os"
	"os/signal"
	"syscall"

	"vpn/pkg/config"
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

	_ = cfg // TODO: use cfg when initializing DB connection

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	sig := <-quit
	log.Info().Str("signal", sig.String()).Msg("shutting down")
}
