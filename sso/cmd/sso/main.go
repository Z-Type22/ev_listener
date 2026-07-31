package main

import (
	"log/slog"
	"os"
	"os/signal"
	"sso/internal/app"
	"sso/internal/config"
	"sso/internal/lib/logger/setup"
	"sso/internal/storage/postgres"
	"syscall"
)

func main() {
	cfg := config.MustLoad()

	log := setup.SetupLogger(cfg.Env)

	log.Info("connecting to database", slog.String("env", cfg.Env))

	storage := postgres.New(cfg.Database)

	log.Info("starting application", slog.String("env", cfg.Env))

	application := app.New(
		log, cfg.GRPC.Port, storage, cfg.PublicKeyPath, cfg.PrivateKeyPath, cfg.AccessTTL, cfg.RefreshTTL,
	)

	go application.GRPCSrv.MustRun()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	sign := <-quit

	log.Info("stopping application", slog.Any("signal", sign))
	log.Info("closing connection to database", slog.Any("signal", sign))

	application.GRPCSrv.Stop()
	storage.Close()
}
