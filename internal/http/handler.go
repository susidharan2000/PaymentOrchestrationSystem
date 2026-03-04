package http

import (
	"net/http"

	intentService "github.com/susidharan/payment-orchestration-system/internal/payment/intent"
	Webhook_ingestor "github.com/susidharan/payment-orchestration-system/internal/webhook_ingestor"
	stripe "github.com/susidharan/payment-orchestration-system/internal/webhook_ingestor/stripe"
)

// create payment
func createPaymentHandler(repo intentService.PaymentRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		intentService.CreatePayment(w, r, repo)
	}
}

// get Payment by ID
func getPayment(repo intentService.PaymentRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		intentService.GetPaymentDetails(w, r, repo)
	}
}

// webhook handler
func stripeWebhook(repo Webhook_ingestor.WebhookRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		stripe.Webhook_ingestor(w, r, repo)
	}
}

// cancel payment
// func cancelPaymentHandler(repo intentService.PaymentRepository) http.HandlerFunc {
// 	return func(w http.ResponseWriter, r *http.Request) {
// 		intentService.CancelPayment(w, r, repo)
// 	}
// }
