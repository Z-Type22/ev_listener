package app

import (
	"context"
	"log/slog"
	"pusher/internal/config"
	"pusher/internal/kafka/producer"
	"pusher/internal/listener/donate"
	"pusher/internal/listener/subscription"
	"time"

	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
)

type App struct {
	client               *ethclient.Client
	donateListener       *donate.DonateListener
	subscriptionListener *subscription.SubscriptionListener

	connectCtx    context.Context
	connectCancel context.CancelFunc

	appCtx    context.Context
	appCancel context.CancelFunc

	writer *producer.KafkaWriter
	log    *slog.Logger
}

func New(
	RPCUrl string,
	clientTimeout time.Duration,
	cfg *config.Config,
	log *slog.Logger,
	writer *producer.KafkaWriter,
) *App {
	connectCtx, cancel := context.WithTimeout(context.Background(), clientTimeout)

	client, err := ethclient.DialContext(connectCtx, RPCUrl)
	if err != nil {
		panic(err)
	}

	donateListener := donate.New(
		client, cfg.ListContracts.DonateContract, log, cfg.PathContracts.DonatePath, writer,
	)
	subscriptionListener := subscription.New(
		client,
		cfg.ListContracts.SubscriptionContract,
		cfg.ListContracts.PlanContract,
		log,
		cfg.PathContracts.SubscriptionPath,
		cfg.PathContracts.PlanPath,
		writer,
	)

	appCtx, appCancel := context.WithCancel(context.Background())

	return &App{
		client: client,

		donateListener:       donateListener,
		subscriptionListener: subscriptionListener,

		connectCtx:    connectCtx,
		connectCancel: cancel,

		appCtx:    appCtx,
		appCancel: appCancel,

		writer: writer,
		log:    log,
	}
}

func (a *App) Listen() {
	const op = "app.Listen"

	log := a.log.With(slog.String("op", op))

	go a.runListener("donate", a.donateListener)
	go a.runListener("subscription", a.subscriptionListener)

	log.Info("all listeners started")

	<-a.appCtx.Done()

	log.Info("application listener stopped")
}

func (a *App) runListener(
	name string, listener interface {
		Listen(context.Context, func(types.Log, context.Context)) error
		Handle(types.Log, context.Context)
	},
) {
	for {
		err := listener.Listen(a.appCtx, listener.Handle)
		if err != nil {
			select {
			case <-a.appCtx.Done():
				return
			default:
				a.log.Error(
					"listener failed, reconnecting", slog.String("listener", name), slog.Any("error", err),
				)
				time.Sleep(5 * time.Second)
			}
		}
	}
}

func (a *App) Stop() {
	a.appCancel()
	a.connectCancel()
	a.client.Close()
	a.writer.Close()
}
