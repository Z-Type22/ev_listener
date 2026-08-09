package server

import (
	"context"
	"log/slog"
	"net/http"
	"rest/internal/config"
	"time"
)

type App struct {
	log             *slog.Logger
	server          *http.Server
	shutdownTimeout time.Duration
}

func New(log *slog.Logger, cfg *config.Config) *App {
	return &App{
		log:             log,
		server:          &http.Server{Addr: cfg.PprofServer.Address},
		shutdownTimeout: cfg.PprofServer.ShutDownTimeout,
	}
}

func (a *App) MustRun() {
	a.log.Info("starting pprof server", slog.String("address", a.server.Addr))

	if err := http.ListenAndServe(a.server.Addr, nil); err != nil {
		a.log.Error("pprof server stopped", slog.String("error", err.Error()))
	}
}

func (a *App) Stop() {
	ctx, cancel := context.WithTimeout(context.Background(), a.shutdownTimeout)
	defer cancel()

	if err := a.server.Shutdown(ctx); err != nil {
		a.log.Error("failed to shutdown pprof server", slog.String("error", err.Error()))
	}

	a.log.Info("pprof server stopped")
}
