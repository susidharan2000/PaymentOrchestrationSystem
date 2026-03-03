package http

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	intentService "github.com/susidharan/payment-orchestration-system/internal/payment/intent"
	Webhook_ingestor "github.com/susidharan/payment-orchestration-system/internal/webhook_ingestor"
)

func NewRouter(paymenttRepo intentService.PaymentRepository, webhookRepo Webhook_ingestor.WebhookRepository) http.Handler {

	req := chi.NewRouter()
	req.Post("/payments", createPaymentHandler(paymenttRepo))    // create Payemnt intent -  client
	req.Post("/webhooks/psp/stripe", stripeWebhook(webhookRepo)) // handles stripe webhook
	//req.Post("/payments/{id}/cancel", cancelPaymentHandler(paymenttRepo)) // cancel payment intent - client

	return req
}
