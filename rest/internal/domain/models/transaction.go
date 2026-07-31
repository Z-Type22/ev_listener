package models

import (
	"time"

	"github.com/shopspring/decimal"
)

type Transaction struct {
	ID             int
	TxHash         string
	UserWallet     string
	MerchantWallet string
	Price          decimal.Decimal
	Type           string
	Payload        map[string]any
	CreatedAt      time.Time
}
