package topic

import (
	"context"
	"net"
	"strconv"
	"time"

	"github.com/segmentio/kafka-go"
)

func CreateTopic(
	address string, topic string, topicTimeout time.Duration, numPartitions int,
) {
	ctx, cancel := context.WithTimeout(context.Background(), topicTimeout)
	defer cancel()

	conn, err := kafka.DialContext(ctx, "tcp", address)
	if err != nil {
		panic(err)
	}
	defer conn.Close()

	controller, err := conn.Controller()
	if err != nil {
		panic(err)
	}

	controllerConn, err := kafka.DialContext(
		ctx, "tcp", net.JoinHostPort(controller.Host, strconv.Itoa(controller.Port)),
	)
	if err != nil {
		panic(err)
	}
	defer controllerConn.Close()

	if err := controllerConn.CreateTopics(
		kafka.TopicConfig{
			Topic:             topic,
			NumPartitions:     numPartitions,
			ReplicationFactor: 1,
		},
	); err != nil {
		panic(err)
	}
}
