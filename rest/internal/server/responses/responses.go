package responses

import (
	"time"

	"github.com/shopspring/decimal"
)

type TransactionResponse struct {
	ID             int             `json:"id"`
	TxHash         string          `json:"tx_hash"`
	UserWallet     string          `json:"user_wallet"`
	MerchantWallet string          `json:"merchant_wallet"`
	Price          decimal.Decimal `json:"price"`
	Type           string          `json:"type"`
	Payload        map[string]any  `json:"payload"`
	CreatedAt      time.Time       `json:"created_at"`
}
