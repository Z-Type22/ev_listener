package producer

import (
	"time"

	"github.com/segmentio/kafka-go"
)

type KafkaWriter struct {
	Writer *kafka.Writer
}

func New(
	address string, topic string, writeTimeout time.Duration, maxAttempts int, batchSize int,
) *KafkaWriter {
	writer := kafka.NewWriter(kafka.WriterConfig{
		Brokers:      []string{address},
		Topic:        topic,
		RequiredAcks: -1,
		MaxAttempts:  maxAttempts,
		BatchSize:    batchSize,
		WriteTimeout: writeTimeout,
		Balancer:     &kafka.RoundRobin{},
	})
	return &KafkaWriter{Writer: writer}
}

func (w *KafkaWriter) Close() {
	w.Writer.Close()
}
