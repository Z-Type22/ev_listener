package main

import (
	"log/slog"
	"os"
	"os/signal"
	"pusher/internal/app"
	"pusher/internal/config"
	"pusher/internal/kafka/producer"
	"pusher/internal/lib/logger/setup"
	"syscall"
)

func main() {
	cfg := config.MustLoad()

	log := setup.SetupLogger(cfg.Env)

	log.Info("connecting to kafka writer", slog.String("env", cfg.Env))
	writer := producer.New(
		cfg.Kafka.Address, cfg.Kafka.Topic, cfg.Kafka.WriteTimeout, cfg.Kafka.MaxAttempts, cfg.Kafka.BatchSize,
	)

	log.Info("starting application", slog.String("env", cfg.Env))
	log.Info("connecting to eth client", slog.String("env", cfg.Env))
	application := app.New(cfg.RPCUrl, cfg.ClientTimeout, cfg, log, writer)

	go application.Listen()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	sign := <-quit

	log.Info("stopping application", slog.Any("signal", sign))
	log.Info("closing connection kafka", slog.Any("signal", sign))

	application.Stop()
	writer.Close()
}
