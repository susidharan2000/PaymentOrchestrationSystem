package webhookingestor

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	stripe "github.com/susidharan/payment-orchestration-system/internal/webhook_ingestor/stripe"
	WebhookRepo "github.com/susidharan/payment-orchestration-system/internal/webhook_ingestor/webhook_repository"
)

func WebhookRouter(w http.ResponseWriter, r *http.Request, repo WebhookRepo.WebhookRepository) {
	psp := chi.URLParam(r, "pspName")
	switch psp {
	case "stripe":
		stripe.Webhook_ingestor(w, r, repo)
	}
}
