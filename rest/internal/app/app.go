package app

import (
	"context"
	"errors"
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

func New(log *slog.Logger, cfg *config.Config, router http.Handler) *App {
	server := &http.Server{
		Addr:         cfg.HTTPServer.Address,
		Handler:      router,
		ReadTimeout:  cfg.HTTPServer.Timeout,
		WriteTimeout: cfg.HTTPServer.Timeout,
		IdleTimeout:  cfg.HTTPServer.IdleTimeout,
	}

	return &App{
		log:             log,
		server:          server,
		shutdownTimeout: cfg.HTTPServer.ShutDownTimeout,
	}
}

func (a *App) MustRun() {
	a.log.Info("starting http-server", slog.String("address", a.server.Addr))

	if err := a.server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		a.log.Error("failed to start server", slog.String("error", err.Error()))
	}
}

func (a *App) Stop() {
	ctx, cancel := context.WithTimeout(context.Background(), a.shutdownTimeout)
	defer cancel()

	if err := a.server.Shutdown(ctx); err != nil {
		a.log.Error("failed to shutdown server", slog.String("error", err.Error()))
	}

	a.log.Info("application stopped")
}
