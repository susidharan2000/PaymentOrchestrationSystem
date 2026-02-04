package http

import (
	"net/http"

	intentService "github.com/susidharan/payment-orchestration-system/internal/payment/intent"
	Webhook_ingestor "github.com/susidharan/payment-orchestration-system/internal/webhook_ingestor"
)

func NewRouter(payemntrepo intentService.PaymentRepository, webhookRepo Webhook_ingestor.WebhookRepository) http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("/payments", createPaymentHandler(payemntrepo))     // create Payemnt intent -  client
	mux.HandleFunc("/webhooks/psp/stripe", stripeWebhook(webhookRepo)) // handles stripe webhook

	return mux
}
