package health

import (
	"context"
	"log/slog"
	"sso/internal/lib/logger/sl"
	"sso/internal/storage/postgres"
)

type Health struct {
	log     *slog.Logger
	storage *postgres.Storage
}

func New(log *slog.Logger, storage *postgres.Storage) *Health {
	return &Health{
		log:     log,
		storage: storage,
	}
}

func (a *Health) Check(ctx context.Context) error {
	const op = "Health.Check"

	log := a.log.With(slog.String("op", op))

	log.Info("check is successfully")

	return nil
}

func (a *Health) Ready(ctx context.Context) error {
	const op = "Health.Ready"

	log := a.log.With(slog.String("op", op))

	log.Info("checking ready")

	if err := a.storage.Check(ctx); err != nil {
		log.Error("is not ready", sl.Err(err))

		return err
	}

	log.Info("is ready")

	return nil
}
