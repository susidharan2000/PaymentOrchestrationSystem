package http

import (
	"net/http"

	intentService "github.com/susidharan/payment-orchestration-system/internal/payment/intent"
)

func NewRouter(repo intentService.PaymentRepository) http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("/payments", createPaymentHandler(repo)) // create Payemnt intent -  client
	mux.HandleFunc("/webhooks/psp/stripe", stripeWebhook()) // handles stripe webhook

	return mux
}
