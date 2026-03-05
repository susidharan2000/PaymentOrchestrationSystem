package domain

import "database/sql"

type PaymentParams struct {
	PaymentId string
	Amount    int64
	Currency  string
	PspRefID  sql.NullString
	PspName   string
}

type PspIntent struct {
	ClientSecret string
	Status       string
}
