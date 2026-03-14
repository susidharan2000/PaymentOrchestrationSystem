package model

import "time"

type RefundRequest struct {
	Amount         int64  `json:"amount"`
	Idempotencykey string `json:"idempotency_key"`
}

type PaymentDetails struct {
	PaymentId string
	Currency  string
	PspName   string
	PspRefID  string
}

type RefundDetails struct {
	RefundID  string    `json:"refund_id"`
	PaymentID string    `json:"payment_id"`
	Amount    int64     `json:"amount"`
	Currency  string    `json:"currency"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
}
