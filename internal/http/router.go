package http

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	paymentrepo "github.com/susidharan/payment-orchestration-system/internal/payment/intent/payment_repository"
	"github.com/susidharan/payment-orchestration-system/internal/psp"
	Webhook_ingestor "github.com/susidharan/payment-orchestration-system/internal/webhook_ingestor"
)

func NewRouter(paymentRepo paymentrepo.PaymentRepository, webhookRepo Webhook_ingestor.WebhookRepository, registry *psp.Registry) http.Handler {

	req := chi.NewRouter()
	req.Post("/payments", createPaymentHandler(paymentRepo, registry)) // create Payemnt intent -  client
	req.Get("/payments/{id}", getPayment(paymentRepo))                 // get payment By id
	//req.Post("/webhooks/psp/stripe", stripeWebhook(webhookRepo)) // handles stripe webhook
	//req.Post("/payments/{id}/cancel", cancelPaymentHandler(paymenttRepo)) // cancel payment intent - client

	return req
}
