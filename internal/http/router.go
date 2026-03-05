package http

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	paymentrepo "github.com/susidharan/payment-orchestration-system/internal/payment/intent/payment_repository"
	Webhook_ingestor "github.com/susidharan/payment-orchestration-system/internal/webhook_ingestor"
)

func NewRouter(paymentRepo paymentrepo.PaymentRepository, webhookRepo Webhook_ingestor.WebhookRepository) http.Handler {

	req := chi.NewRouter()
	req.Post("/payments", createPaymentHandler(paymentRepo)) // create Payemnt intent -  client
	req.Get("/payments/{id}", getPayment(paymentRepo))       // get payment By id
	//req.Post("/webhooks/psp/stripe", stripeWebhook(webhookRepo)) // handles stripe webhook
	//req.Post("/payments/{id}/cancel", cancelPaymentHandler(paymenttRepo)) // cancel payment intent - client

	return req
}
