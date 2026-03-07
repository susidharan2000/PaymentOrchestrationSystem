package domain

import "database/sql"

type PaymentStatus string

const (
	StatusProcessing PaymentStatus = "PROCESSING"
	StatusSucceeded  PaymentStatus = "SUCCEEDED"
	StatusFailed     PaymentStatus = "FAILED"
)

type PaymentParams struct {
	PaymentId string
	Amount    int64
	Currency  string
	PspRefID  sql.NullString
	PspName   string
}

type PspIntent struct {
	ClientSecret string
	Status       PaymentStatus
}
