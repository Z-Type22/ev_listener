package subscription

import (
	"context"
	"log/slog"
	"math/big"
	"time"

	"pusher/internal/clients/kafka/producer"
	"pusher/internal/lib/contracts"
	"pusher/internal/listener"
	"pusher/internal/listener/plan"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/segmentio/kafka-go"
	"github.com/shopspring/decimal"
)

type SubscriptionListener struct {
	*listener.BaseListener
	*producer.KafkaWriter
	plan        *plan.PlanManager
	handlers    map[string]func(types.Log, context.Context)
	contractABI abi.ABI
}

type Plan struct {
	Hash      string    `json:"hash"`
	Period    uint64    `json:"period"`
	Token     string    `json:"token"`
	Status    uint8     `json:"status"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	URI       string    `json:"uri"`
}

type BaseSubscriptionEvent struct {
	MerchantAddress string          `json:"merchant_address"`
	UserAddress     string          `json:"user_address"`
	TxHash          string          `json:"tx_hash"`
	Price           decimal.Decimal `json:"price"`
}

type ActivatedEvent struct {
	BaseSubscriptionEvent

	Plan         *Plan     `json:"plan,omitempty"`
	PlanHash     string    `json:"plan_hash"`
	NextChargeAt time.Time `json:"next_charge_at"`
}

type ChargedEvent struct {
	BaseSubscriptionEvent

	NextChargeAt time.Time `json:"next_charge_at"`
}

type DeactivatedEvent struct {
	BaseSubscriptionEvent

	Reason string `json:"reason"`
}

type RetryScheduledEvent struct {
	BaseSubscriptionEvent

	RetryAt time.Time `json:"retry_at"`
}

// ABI
type SubscriptionActivated struct {
	NextChargeAt *big.Int
}

type SubscriptionCancelled struct {
	Reason uint8
}

type SubscriptionCharged struct {
	Amount       *big.Int
	NextChargeAt *big.Int
}

type SubscriptionFailedFinal struct {
	Reason uint8
}

type SubscriptionRetryScheduled struct {
	RetryAt    *big.Int
	RetryCount uint16
}

func New(
	client *ethclient.Client,
	addressSub string,
	addressPlan string,
	log *slog.Logger,
	pathSub string,
	pathPlan string,
	writer *producer.KafkaWriter,
) *SubscriptionListener {
	baseListener := listener.New(client, addressSub, log)

	planABI, err := contracts.LoadABI(pathPlan)
	if err != nil {
		panic(err)
	}

	plan := &plan.PlanManager{
		Address:     common.HexToAddress(addressPlan),
		ContractABI: planABI,
		Client:      client,
	}

	subscriptionABI, err := contracts.LoadABI(pathSub)
	if err != nil {
		panic(err)
	}

	s := &SubscriptionListener{
		BaseListener: baseListener,
		KafkaWriter:  writer,
		plan:         plan,
		handlers:     make(map[string]func(types.Log, context.Context)),
		contractABI:  subscriptionABI,
	}

	eventSubscriptionActivated := subscriptionABI.Events["SubscriptionActivated"]
	eventSubscriptionCharged := subscriptionABI.Events["SubscriptionCharged"]
	eventSubscriptionCancelled := subscriptionABI.Events["SubscriptionCancelled"]
	eventSubscriptionFailedFinal := subscriptionABI.Events["SubscriptionFailedFinal"]
	eventSubscriptionRetryScheduled := subscriptionABI.Events["SubscriptionRetryScheduled"]

	s.handlers[eventSubscriptionActivated.ID.Hex()] = s.HandleSubscriptionActivated
	s.handlers[eventSubscriptionCharged.ID.Hex()] = s.HandleSubscriptionCharged
	s.handlers[eventSubscriptionCancelled.ID.Hex()] = s.HandleSubscriptionCancelled
	s.handlers[eventSubscriptionFailedFinal.ID.Hex()] = s.HandleSubscriptionFailedFinal
	s.handlers[eventSubscriptionRetryScheduled.ID.Hex()] = s.HandleSubscriptionRetryScheduled

	return s
}

func (s *SubscriptionListener) HandleSubscriptionActivated(vLog types.Log, ctx context.Context) {
	const op = "listener.subscription.HandleSubscriptionActivated"

	log := s.Log.With(slog.String("op", op))

	var subscriptionActivated SubscriptionActivated

	title := "SubscriptionActivated"

	err := s.contractABI.UnpackIntoInterface(
		&subscriptionActivated, title, vLog.Data,
	)
	if err != nil {
		log.Error("failed unpack interface", slog.Any("err", err))
		return
	}

	user := common.HexToAddress(vLog.Topics[1].Hex())
	merchant := common.HexToAddress(vLog.Topics[3].Hex())
	planHash := vLog.Topics[2].Hex()

	planData, err := s.plan.GetPlan(ctx, common.HexToHash(planHash))
	if err != nil {
		log.Error("failed get plan", slog.Any("err", err))
		return
	}

	price := decimal.NewFromBigInt(planData.Price, 0).Div(
		decimal.NewFromInt(1_000_000_000_000_000_000),
	)

	message := ActivatedEvent{
		BaseSubscriptionEvent: BaseSubscriptionEvent{
			MerchantAddress: merchant.Hex(),
			UserAddress:     user.Hex(),
			TxHash:          vLog.TxHash.Hex(),
			Price:           price,
		},

		PlanHash: planHash,

		Plan: &Plan{
			Hash:      common.BytesToHash(planData.Hash[:]).Hex(),
			Period:    uint64(planData.Period),
			Token:     planData.Token.Hex(),
			Status:    planData.Status,
			CreatedAt: time.Unix(planData.CreatedAt.Int64(), 0),
			UpdatedAt: time.Unix(planData.UpdatedAt.Int64(), 0),
			URI:       planData.Uri,
		},

		NextChargeAt: time.Unix(subscriptionActivated.NextChargeAt.Int64(), 0),
	}

	data, err := s.MakePayload(title, message)
	if err != nil {
		log.Error("failed marshal subscription activated event", slog.Any("err", err))
		return
	}

	if err := s.Writer.WriteMessages(ctx, kafka.Message{Value: data}); err != nil {
		log.Error("failed write kafka message", slog.Any("error", err))
	}

	log.Info("subscription activated event published", slog.String("tx_hash", message.TxHash))
}

func (s *SubscriptionListener) HandleSubscriptionCharged(vLog types.Log, ctx context.Context) {
	const op = "listener.subscription.HandleSubscriptionCharged"

	log := s.Log.With(slog.String("op", op))

	var subscriptionCharged SubscriptionCharged

	title := "SubscriptionCharged"

	err := s.contractABI.UnpackIntoInterface(
		&subscriptionCharged, title, vLog.Data,
	)
	if err != nil {
		log.Error("failed unpack interface", slog.Any("err", err))
		return
	}

	user := common.HexToAddress(vLog.Topics[1].Hex())
	planHash := vLog.Topics[2].Hex()

	planData, err := s.plan.GetPlan(ctx, common.HexToHash(planHash))

	price := decimal.NewFromBigInt(planData.Price, 0).Div(
		decimal.NewFromInt(1_000_000_000_000_000_000),
	)

	message := ChargedEvent{
		BaseSubscriptionEvent: BaseSubscriptionEvent{
			MerchantAddress: planData.Merchant.Hex(),
			UserAddress:     user.Hex(),
			TxHash:          vLog.TxHash.Hex(),
			Price:           price,
		},

		NextChargeAt: time.Unix(subscriptionCharged.NextChargeAt.Int64(), 0),
	}

	data, err := s.MakePayload(title, message)
	if err != nil {
		log.Error("failed marshal subscription charged event", slog.Any("err", err))
		return
	}

	if err := s.Writer.WriteMessages(ctx, kafka.Message{Value: data}); err != nil {
		log.Error("failed write kafka message", slog.Any("error", err))
	}

	log.Info("subscription charged event published", slog.String("tx_hash", message.TxHash))
}

func (s *SubscriptionListener) HandleSubscriptionCancelled(vLog types.Log, ctx context.Context) {
	const op = "listener.subscription.HandleSubscriptionCancelled"

	log := s.Log.With(slog.String("op", op))

	var event SubscriptionCancelled

	title := "SubscriptionCancelled"

	err := s.contractABI.UnpackIntoInterface(
		&event, title, vLog.Data,
	)
	if err != nil {
		log.Error("failed unpack interface", slog.Any("err", err))
		return
	}

	s.handleSubscriptionDeactivated(vLog, ctx, title)
}

func (s *SubscriptionListener) HandleSubscriptionFailedFinal(vLog types.Log, ctx context.Context) {
	const op = "listener.subscription.HandleSubscriptionFailedFinal"

	log := s.Log.With(slog.String("op", op))

	var event SubscriptionFailedFinal

	title := "SubscriptionFailedFinal"

	err := s.contractABI.UnpackIntoInterface(
		&event, title, vLog.Data,
	)
	if err != nil {
		log.Error("failed unpack interface", slog.Any("err", err))
		return
	}

	s.handleSubscriptionDeactivated(vLog, ctx, title)
}

func (s *SubscriptionListener) HandleSubscriptionRetryScheduled(vLog types.Log, ctx context.Context) {
	const op = "listener.subscription.HandleSubscriptionFailedFinal"

	log := s.Log.With(slog.String("op", op))

	var subscriptionRetryScheduled SubscriptionRetryScheduled

	title := "SubscriptionRetryScheduled"

	err := s.contractABI.UnpackIntoInterface(
		&subscriptionRetryScheduled, title, vLog.Data,
	)
	if err != nil {
		log.Error("failed unpack interface", slog.Any("err", err))
		return
	}

	user := common.HexToAddress(vLog.Topics[1].Hex())
	planHash := vLog.Topics[2].Hex()

	planData, err := s.plan.GetPlan(ctx, common.HexToHash(planHash))
	if err != nil {
		log.Error("failed get plan", slog.Any("err", err))
		return
	}

	price := decimal.NewFromBigInt(planData.Price, 0).Div(
		decimal.NewFromInt(1_000_000_000_000_000_000),
	)

	message := RetryScheduledEvent{
		BaseSubscriptionEvent: BaseSubscriptionEvent{
			MerchantAddress: planData.Merchant.Hex(),
			UserAddress:     user.Hex(),
			TxHash:          vLog.TxHash.Hex(),
			Price:           price,
		},

		RetryAt: time.Unix(subscriptionRetryScheduled.RetryAt.Int64(), 0),
	}

	data, err := s.MakePayload(title, message)
	if err != nil {
		log.Error("failed marshal subscription failed final event", slog.Any("err", err))
		return
	}

	if err := s.Writer.WriteMessages(ctx, kafka.Message{Value: data}); err != nil {
		log.Error("failed write kafka message", slog.Any("error", err))
		return
	}

	log.Info("subscription retry scheduled event published", slog.String("tx_hash", message.TxHash))
}

// Helper
func (s *SubscriptionListener) handleSubscriptionDeactivated(
	vLog types.Log, ctx context.Context, eventName string,
) {
	const op = "listener.subscription.handleSubscriptionDeactivated"

	log := s.Log.With(slog.String("op", op), slog.String("event", eventName))

	user := common.HexToAddress(vLog.Topics[1].Hex())
	planHash := vLog.Topics[2].Hex()

	planData, err := s.plan.GetPlan(ctx, common.HexToHash(planHash))
	if err != nil {
		log.Error("failed get plan", slog.Any("err", err))
		return
	}

	price := decimal.NewFromBigInt(planData.Price, 0).Div(
		decimal.NewFromInt(1_000_000_000_000_000_000),
	)

	message := DeactivatedEvent{
		BaseSubscriptionEvent: BaseSubscriptionEvent{
			MerchantAddress: planData.Merchant.Hex(),
			UserAddress:     user.Hex(),
			TxHash:          vLog.TxHash.Hex(),
			Price:           price,
		},

		Reason: eventName,
	}

	data, err := s.MakePayload(eventName, message)
	if err != nil {
		log.Error("failed marshal subscription deactivated event", slog.Any("err", err))
		return
	}

	if err := s.Writer.WriteMessages(ctx, kafka.Message{Value: data}); err != nil {
		log.Error("failed write kafka message", slog.Any("error", err))
		return
	}

	log.Info("subscription deactivated event published", slog.String("tx_hash", message.TxHash))
}

func (s *SubscriptionListener) Handle(vLog types.Log, ctx context.Context) {
	const op = "listener.subscription.Handle"

	log := s.Log.With(slog.String("op", op))

	if len(vLog.Topics) == 0 {
		log.Error("log has no topics")
		return
	}

	topic := vLog.Topics[0].Hex()
	handler, ok := s.handlers[topic]
	if !ok {
		log.Debug("event handler not found", slog.String("topic", topic))
		return
	}

	handler(vLog, ctx)
}
