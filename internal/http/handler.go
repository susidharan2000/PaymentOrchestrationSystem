package http

import (
	"net/http"

	paymentIntent "github.com/susidharan/payment-orchestration-system/internal/payment/intent/paymentService"
	paymentrepo "github.com/susidharan/payment-orchestration-system/internal/payment/intent/payment_repository"
	"github.com/susidharan/payment-orchestration-system/internal/psp"
	// Webhook_Repo "github.com/susidharan/payment-orchestration-system/internal/webhook_ingestor/webhook_repository"
	// Webhook_ingestor "github.com/susidharan/payment-orchestration-system/internal/webhook_ingestor/webhook_router"
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
// func WebhookHandler(repo Webhook_Repo.WebhookRepository) http.HandlerFunc {
// 	return func(w http.ResponseWriter, r *http.Request) {
// 		Webhook_ingestor.WebhookRouter(w, r, repo)
// 	}
// }

// cancel payment
// func cancelPaymentHandler(repo intentService.PaymentRepository) http.HandlerFunc {
// 	return func(w http.ResponseWriter, r *http.Request) {
// 		intentService.CancelPayment(w, r, repo)
// 	}
// }
