package http

import (
	"net/http"

	paymentIntent "github.com/susidharan/payment-orchestration-system/internal/payment/intent/paymentService"
	paymentrepo "github.com/susidharan/payment-orchestration-system/internal/payment/intent/payment_repository"
	"github.com/susidharan/payment-orchestration-system/internal/psp"
	// Webhook_ingestor "github.com/susidharan/payment-orchestration-system/internal/webhook_ingestor"
	// stripe "github.com/susidharan/payment-orchestration-system/internal/webhook_ingestor/stripe"
)

// create payment
func createPaymentHandler(repo paymentrepo.PaymentRepository, registry *psp.Registry) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		paymentIntent.CreatePayment(w, r, repo, registry)
	}
}

// get Payment by ID
func getPayment(repo paymentrepo.PaymentRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		paymentIntent.GetPaymentDetails(w, r, repo)
	}
}

// webhook handler
// func stripeWebhook(repo Webhook_ingestor.WebhookRepository) http.HandlerFunc {
// 	return func(w http.ResponseWriter, r *http.Request) {
// 		stripe.Webhook_ingestor(w, r, repo)
// 	}
// }

// cancel payment
// func cancelPaymentHandler(repo intentService.PaymentRepository) http.HandlerFunc {
// 	return func(w http.ResponseWriter, r *http.Request) {
// 		intentService.CancelPayment(w, r, repo)
// 	}
// }
