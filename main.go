package main

import (
	"os"
	"os/signal"
	"syscall"

	"vpn/pkg/logger"

	"github.com/rs/zerolog/log"
)

func main() {
	logger.Setup()

	log.Info().Msg("starting vpn server")

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	sig := <-quit
	log.Info().Str("signal", sig.String()).Msg("shutting down")
}
