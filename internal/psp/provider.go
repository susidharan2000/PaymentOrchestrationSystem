package psp

import "github.com/susidharan/payment-orchestration-system/internal/domain"

// type PaymentStatus string

// const (
// 	StatusProcessing PaymentStatus = "PROCESSING"
// 	StatusSucceeded  PaymentStatus = "SUCCEEDED"
// 	StatusFailed     PaymentStatus = "FAILED"
// )

type PSP interface {
	CreatePaymentIntent(paymentDetails domain.PaymentParams) (string, string, error)
	GetPaymentIntent(pspRefID string) (domain.PspIntent, bool, error)
	CreateRefund(pspPaymentRefID string, amount int64, idempotencyKey string) (string, error)
}
