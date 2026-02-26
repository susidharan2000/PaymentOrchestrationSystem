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
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		intentService.CreatePayment(w, r, repo)
	}
}

// cancel payment
func cancelPaymentHandler(repo intentService.PaymentRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		intentService.CancelPayment(w, r, repo)
	}
}

func stripeWebhook(repo Webhook_ingestor.WebhookRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		stripe.Webhook_ingestor(w, r, repo)
	}
}
