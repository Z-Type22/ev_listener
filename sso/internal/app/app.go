package app

import (
	"log/slog"
	grpcapp "sso/internal/app/grpc"
	"sso/internal/services/auth"
	"sso/internal/services/health"
	"sso/internal/storage/postgres"
	"time"
)

type App struct {
	GRPCSrv *grpcapp.App
}

func New(
	log *slog.Logger,
	port int,
	storage *postgres.Storage,
	publicKeyPath string,
	privateKeyPath string,
	accessTTL time.Duration,
	refreshTTL time.Duration,
) *App {
	authService := auth.New(
		log, storage, storage, storage, publicKeyPath, privateKeyPath, accessTTL, refreshTTL,
	)
	healthService := health.New(log, storage)

	grpcapp := grpcapp.New(log, authService, healthService, port)

	return &App{GRPCSrv: grpcapp}
}
