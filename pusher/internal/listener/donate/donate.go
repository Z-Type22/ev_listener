package donate

import (
	"context"
	"log/slog"
	"math/big"
	"pusher/internal/kafka/producer"
	"pusher/internal/lib/contracts"
	"pusher/internal/listener"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/segmentio/kafka-go"
	"github.com/shopspring/decimal"
)

type DonateListener struct {
	*listener.BaseListener
	*producer.KafkaWriter
	handlers    map[string]func(types.Log, context.Context)
	ContractABI abi.ABI
}

type DonateEvent struct {
	TxHash           string          `json:"tx_hash"`
	MerchantAddress  string          `json:"merchant_address"`
	RecipientAddress string          `json:"recipient_address"`
	Amount           decimal.Decimal `json:"amount"`
}

// ABI
type DonationProcessed struct {
	Token       common.Address
	GrossAmount *big.Int
	NetAmount   *big.Int
	Metadata    [32]byte
	Timestamp   *big.Int
	ModuleId    [32]byte
}

func New(
	client *ethclient.Client, address string, log *slog.Logger, path string, writer *producer.KafkaWriter,
) *DonateListener {
	baseListener := listener.New(client, address, log)

	donateABI, err := contracts.LoadABI(path)
	if err != nil {
		panic(err)
	}

	d := &DonateListener{
		BaseListener: baseListener,
		KafkaWriter:  writer,
		ContractABI:  donateABI,
		handlers:     make(map[string]func(types.Log, context.Context)),
	}

	event := donateABI.Events["DonationProcessed"]
	d.handlers[event.ID.Hex()] = d.HandleDonationProcessed

	return d
}

func (d *DonateListener) HandleDonationProcessed(vLog types.Log, ctx context.Context) {
	const op = "listener.donate.HandleDonationProcessed"

	log := d.Log.With(slog.String("op", op))

	var donationProcessed DonationProcessed

	title := "DonationProcessed"

	err := d.ContractABI.UnpackIntoInterface(
		&donationProcessed, title, vLog.Data,
	)
	if err != nil {
		log.Error("failed unpack interface", slog.Any("err", err))
		return
	}

	wei := decimal.NewFromBigInt(donationProcessed.GrossAmount, 0)
	ether := wei.Div(decimal.NewFromInt(1_000_000_000_000_000_000))

	txHash := vLog.TxHash.Hex()
	donor := common.HexToAddress(vLog.Topics[2].Hex()).Hex()
	recipient := common.HexToAddress(vLog.Topics[3].Hex()).Hex()

	message := DonateEvent{
		TxHash:           txHash,
		MerchantAddress:  donor,
		RecipientAddress: recipient,
		Amount:           ether,
	}

	data, err := d.MakePayload(title, message)
	if err != nil {
		log.Error("failed marshal subscription charged event", slog.Any("err", err))
		return
	}

	if err := d.Writer.WriteMessages(ctx, kafka.Message{Value: data}); err != nil {
		log.Error("failed write kafka message", slog.Any("error", err))
	}

	log.Info("donate event published", slog.String("tx_hash", message.TxHash))
}

func (d *DonateListener) Handle(vLog types.Log, ctx context.Context) {
	const op = "listener.donate.Handle"

	log := d.Log.With(slog.String("op", op))

	if len(vLog.Topics) == 0 {
		log.Error("log has no topics")
		return
	}

	topic := vLog.Topics[0].Hex()
	handler, ok := d.handlers[topic]
	if !ok {
		log.Debug("event handler not found", slog.String("topic", topic))
		return
	}

	handler(vLog, ctx)
}
