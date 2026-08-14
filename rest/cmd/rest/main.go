package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	_ "rest/docs"
	"rest/internal/app"
	"rest/internal/clients/kafka/consumer"
	"rest/internal/clients/kafka/topic"
	ssogrpc "rest/internal/clients/sso/grpc"
	"rest/internal/config"
	"rest/internal/lib/logger/setup"
	custom_middleware "rest/internal/middleware"
	"rest/internal/server/handlers/gettran"
	"rest/internal/server/handlers/gettrans"
	"rest/internal/server/handlers/healthcheck"
	"rest/internal/server/handlers/healthready"
	"rest/internal/server/handlers/login"
	"rest/internal/server/handlers/logout"
	"rest/internal/server/handlers/refresh"
	"rest/internal/server/handlers/register"
	"rest/internal/storage/postgres"
	pprofApp "rest/profiling/server"
	"syscall"
	"time"

	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
	httpSwagger "github.com/swaggo/http-swagger/v2"
)

// @title Transactions REST API
// @version 1.0
// @description REST API for registration, authentication, and transaction retrieval.
// @host localhost:8000
// @BasePath /
// @schemes http
// @accept json
// @produce json
// @securityDefinitions.apikey AccessTokenCookie
// @in header
// @name Cookie
// @description Set this value to `access_token=<JWT>` for protected endpoints.
//
//go:generate go run github.com/swaggo/swag/cmd/swag@v1.16.6 init -g cmd/rest/main.go -d ../.. -o ../../docs --parseInternal
func main() {
	cfg := config.MustLoad()

	log := setup.SetupLogger(cfg.Env)

	log.Info("connecting to sso", slog.String("env", cfg.Env))

	clientSSO := ssogrpc.New(
		context.Background(), log, cfg.Clients.SSO.Address, cfg.Clients.SSO.Timeout, cfg.Clients.SSO.RetriesCount,
	)

	log.Info("connecting to database", slog.String("env", cfg.Env))

	storage := postgres.New(cfg.Database, log)

	log.Info("starting application", slog.String("env", cfg.Env))

	log.Info("creating topics", slog.String("env", cfg.Env))
	topic.CreateTopic(cfg.Clients.Kafka.Address, cfg.Clients.Kafka.Topic, cfg.Clients.Kafka.TopicTimeout, cfg.Clients.Kafka.NumPartitions)

	log.Info("connecting to kafka reader", slog.String("env", cfg.Env))
	reader := consumer.New(cfg.Clients.Kafka.Address, cfg.Clients.Kafka.Topic, cfg.Clients.Kafka.GroupID, storage, log)

	router := chi.NewRouter()

	router.Use(middleware.RequestID)
	router.Use(middleware.Logger)
	router.Use(middleware.Recoverer)
	router.Use(middleware.URLFormat)
	router.Use(middleware.Timeout(cfg.HTTPServer.Timeout * time.Second))

	router.Get("/swagger/*", httpSwagger.Handler(httpSwagger.URL("/swagger/doc.json")))

	router.Get("/health/check", healthcheck.New(log))
	router.Get("/health/ready", healthready.New(log, storage, clientSSO, reader))

	router.Route("/v1/transactions", func(r chi.Router) {
		r.Group(func(r chi.Router) {
			r.Use(custom_middleware.Auth(log, cfg.PublicKeyPath))

			r.Get("/{id}", gettran.New(log, storage))
			r.Get("/", gettrans.New(log, storage))
		})

		r.Route("/auth", func(r chi.Router) {
			r.Post("/register", register.New(log, clientSSO))
			r.Post("/login", login.New(log, clientSSO))
			r.Post("/refresh", refresh.New(log, clientSSO))
			r.Post("/logout", logout.New(log, clientSSO))
		})
	})

	application := app.New(log, cfg, router)
	pprofServer := pprofApp.New(log, cfg)

	go application.MustRun()
	go pprofServer.MustRun()
	go reader.MustRun()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	sign := <-quit

	log.Info("stopping application", slog.Any("signal", sign))
	application.Stop()

	log.Info("stopping pprof-server", slog.Any("signal", sign))
	pprofServer.Stop()

	log.Info("closing connection to sso", slog.Any("signal", sign))
	clientSSO.Close()

	log.Info("closing connection to database", slog.Any("signal", sign))
	storage.Close()

	log.Info("closing connection to kafka", slog.Any("signal", sign))
	reader.Close()
}
