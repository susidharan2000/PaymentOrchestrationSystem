package http

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	paymentrepo "github.com/susidharan/payment-orchestration-system/internal/payment/intent/payment_repository"
	"github.com/susidharan/payment-orchestration-system/internal/psp"
	Webhook_Repo "github.com/susidharan/payment-orchestration-system/internal/webhook_ingestor/webhook_repository"
)

func NewRouter(paymentRepo paymentrepo.PaymentRepository, webhookRepo Webhook_Repo.WebhookRepository, registry *psp.Registry) http.Handler {

	req := chi.NewRouter()
	req.Post("/payments", createPaymentHandler(paymentRepo, registry)) // create Payemnt intent -  client
	req.Get("/payments/{id}", getPayment(paymentRepo))                 // get payment By id
	req.Post("/webhooks/psp/{pspName}", WebhookHandler(webhookRepo))   // handles stripe webhook
	//req.Post("/payments/{id}/cancel", cancelPaymentHandler(paymenttRepo)) // cancel payment intent - client

	return req
}
