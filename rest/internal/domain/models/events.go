package models

import (
	"encoding/json"
	"time"

	"github.com/shopspring/decimal"
)

type EventEnvelope struct {
	Title string          `json:"title"`
	Data  json.RawMessage `json:"data"`
}

type BaseEvent struct {
	MerchantAddress string          `json:"merchant_address"`
	UserAddress     string          `json:"user_address"`
	TxHash          string          `json:"tx_hash"`
	Price           decimal.Decimal `json:"price"`
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

type DonateEvent struct {
	TxHash           string          `json:"tx_hash"`
	MerchantAddress  string          `json:"merchant_address"`
	RecipientAddress string          `json:"recipient_address"`
	Amount           decimal.Decimal `json:"Amount"`
}

type ActivatedEvent struct {
	BaseEvent

	PlanHash string `json:"plan_hash"`
	Plan     *Plan  `json:"plan"`

	NextChargeAt time.Time `json:"next_charge_at"`
}

type ChargedEvent struct {
	BaseEvent

	NextChargeAt time.Time `json:"next_charge_at"`
}

type DeactivatedEvent struct {
	BaseEvent

	Reason string `json:"reason"`
}

type RetryScheduledEvent struct {
	BaseEvent

	RetryAt time.Time `json:"retry_at"`
}
