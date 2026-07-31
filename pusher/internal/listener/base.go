package listener

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
)

type BaseListener struct {
	Client  *ethclient.Client
	Address common.Address
	Log     *slog.Logger
}

func New(client *ethclient.Client, address string, log *slog.Logger) *BaseListener {
	return &BaseListener{
		Client:  client,
		Address: common.HexToAddress(address),
		Log:     log,
	}
}

func (l *BaseListener) Listen(ctx context.Context, handler func(types.Log, context.Context)) error {
	const op = "listener.base.Listen"

	log := l.Log.With(slog.String("op", op), slog.String("addr", l.Address.Hex()))

	logs := make(chan types.Log)

	query := ethereum.FilterQuery{Addresses: []common.Address{l.Address}}

	filterLogs, err := l.Client.SubscribeFilterLogs(ctx, query, logs)
	if err != nil {
		return err
	}

	log.Info("listening to the contract")

	for {
		select {
		case err := <-filterLogs.Err():
			return err
		case vLog := <-logs:
			handler(vLog, ctx)
		case <-ctx.Done():
			return nil
		}
	}
}

func (b *BaseListener) MakePayload(title string, data any) ([]byte, error) {
	payload := struct {
		Title string `json:"title"`
		Data  any    `json:"data"`
	}{
		Title: title,
		Data:  data,
	}

	message, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", "failed marshal event", err)
	}

	return message, nil
}
