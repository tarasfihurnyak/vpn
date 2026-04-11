package main

import (
	"context"
	"errors"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog/log"

	sqlcdb "vpn/internal/db/sqlc"
	"vpn/internal/peer"
	"vpn/internal/server"
	"vpn/internal/user"
	"vpn/pkg/config"
	"vpn/pkg/db"
	"vpn/pkg/logger"
)

func main() {
	logger.Setup()

	cfg, err := config.Load(".env")
	if err != nil {
		log.Fatal().Err(err).Msg("load config")
	}

	log.Info().Msg("starting vpn server")

	sqlDB, err := db.Connect(cfg.DB.DSN())
	if err != nil {
		log.Fatal().Err(err).Msg("connect db")
	}
	defer func() {
		if err := sqlDB.Close(); err != nil {
			log.Error().Err(err).Msg("close db")
		}
	}()

	n, err := db.MigrateUp(sqlDB)
	if err != nil {
		log.Fatal().Err(err).Msg("run migrations")
	}
	log.Info().Int("count", n).Msg("migrations applied")

	pool, err := pgxpool.New(context.Background(), cfg.DB.DSN())
	if err != nil {
		log.Fatal().Err(err).Msg("open pgxpool")
	}
	defer pool.Close()

	q := sqlcdb.New(pool)
	userHandler := user.NewHandler(user.NewService(q))
	peerHandler := peer.NewHandler(peer.NewService(q))
	addr := ":8080"
	httpSrv := &http.Server{
		Addr:         addr,
		Handler:      server.NewHTTP(userHandler, peerHandler),
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	go func() {
		log.Info().Str("addr", addr).Msg("http server listening")
		if err := httpSrv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatal().Err(err).Msg("http server error")
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Info().Msg("shutting down")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := httpSrv.Shutdown(ctx); err != nil {
		log.Error().Err(err).Msg("shutdown error")
	}
}
