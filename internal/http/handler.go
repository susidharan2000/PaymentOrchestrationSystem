package http

import (
	"net/http"

	paymentIntent "github.com/susidharan/payment-orchestration-system/internal/payment/intent/paymentService"
	paymentrepo "github.com/susidharan/payment-orchestration-system/internal/payment/intent/payment_repository"
	"github.com/susidharan/payment-orchestration-system/internal/psp"
	refund_repo "github.com/susidharan/payment-orchestration-system/internal/refund/intent/refund_repository"
	refundIntent "github.com/susidharan/payment-orchestration-system/internal/refund/intent/refund_service"
	Webhook_Repo "github.com/susidharan/payment-orchestration-system/internal/webhook_ingestor/webhook_repository"
	Webhook_ingestor "github.com/susidharan/payment-orchestration-system/internal/webhook_ingestor/webhook_router"
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

// Create refund
func refundPayment(repo refund_repo.RefundRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		refundIntent.CreateRefundIntent(w, r, repo)
	}
}

// webhook handler
func WebhookHandler(repo Webhook_Repo.WebhookRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		Webhook_ingestor.WebhookRouter(w, r, repo)
	}
}

// cancel payment
// func cancelPaymentHandler(repo intentService.PaymentRepository) http.HandlerFunc {
// 	return func(w http.ResponseWriter, r *http.Request) {
// 		intentService.CancelPayment(w, r, repo)
// 	}
// }
