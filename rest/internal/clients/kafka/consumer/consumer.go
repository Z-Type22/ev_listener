package consumer

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"rest/internal/domain/models"
	as_consumer "rest/internal/lib/consumer"
	"rest/internal/storage/postgres"

	"github.com/segmentio/kafka-go"
)

type KafkaReader struct {
	reader  *kafka.Reader
	address string
	storage *postgres.Storage
	ctx     context.Context
	cancel  context.CancelFunc
	log     *slog.Logger
}

func New(address string, topic string, groupId string, storage *postgres.Storage, log *slog.Logger) *KafkaReader {
	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers: []string{address}, Topic: topic, GroupID: groupId,
	})

	ctx, cancel := context.WithCancel(context.Background())

	return &KafkaReader{
		reader:  reader,
		address: address,
		storage: storage,
		ctx:     ctx,
		cancel:  cancel,
		log:     log,
	}
}

func (r *KafkaReader) Check(ctx context.Context) error {
	conn, err := kafka.DialContext(ctx, "tcp", r.address)
	if err != nil {
		return err
	}
	defer conn.Close()

	_, err = conn.ReadPartitions()

	return err
}

func (r *KafkaReader) MustRun() {
	for {
		msg, err := r.reader.ReadMessage(r.ctx)
		if err != nil {
			if r.ctx.Err() != nil {
				r.log.Info("kafka reader stopped")
				return
			}

			r.log.Error("kafka read failed", slog.Any("error", err))
			return
		}

		id, err := r.Process(msg)
		if err != nil {
			r.log.Error("failed set transaction", slog.Any("error", err))
		}

		r.log.Info("transaction successfully saved", slog.Int("id", id))
	}
}

func (r *KafkaReader) Process(msg kafka.Message) (int, error) {
	var envelope models.EventEnvelope

	if err := json.Unmarshal(msg.Value, &envelope); err != nil {
		return 0, err
	}

	switch envelope.Title {
	case "DonationProcessed":
		return as_consumer.ProcessEvent(r.ctx, envelope, r.storage.SetTransactionDonate)

	case "SubscriptionActivated":
		return as_consumer.ProcessEvent(r.ctx, envelope, r.storage.SetTransactionSubscriptionActivated)

	case "SubscriptionCharged":
		return as_consumer.ProcessEvent(r.ctx, envelope, r.storage.SetTransactionSubscriptionCharged)

	case "SubscriptionCancelled":
		return as_consumer.ProcessEvent(r.ctx, envelope, r.storage.SetTransactionSubscriptionCancelled)

	case "SubscriptionFailedFinal":
		return as_consumer.ProcessEvent(r.ctx, envelope, r.storage.SetTransactionSubscriptionFailedFinal)

	case "SubscriptionRetryScheduled":
		return as_consumer.ProcessEvent(r.ctx, envelope, r.storage.SetTransactionSubscriptionRetryScheduled)
	}

	return 0, fmt.Errorf("unsupported event type: %s", envelope.Title)
}

func (r *KafkaReader) Close() {
	r.cancel()
	r.reader.Close()
}
