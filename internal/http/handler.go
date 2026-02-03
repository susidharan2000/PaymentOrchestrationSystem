package http

import (
	"net/http"

	intentService "github.com/susidharan/payment-orchestration-system/internal/payment/intent"
	stripe "github.com/susidharan/payment-orchestration-system/internal/webhook_ingestor/stripe"
)

func createPaymentHandler(repo intentService.PaymentRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		intentService.CreatePayment(w, r, repo)
	}
}

func stripeWebhook() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		stripe.Webhook_ingestor(w, r)
	}
}
