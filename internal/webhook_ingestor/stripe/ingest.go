package stripe

import (
	"io"
	"log"
	"net/http"
	"os"

	"github.com/stripe/stripe-go/v78/webhook"
	model "github.com/susidharan/payment-orchestration-system/internal/webhook_ingestor/model"
	WebhookRepo "github.com/susidharan/payment-orchestration-system/internal/webhook_ingestor/webhook_repository"
)

func Webhook_ingestor(w http.ResponseWriter, r *http.Request, repo WebhookRepo.WebhookRepository) {
	defer r.Body.Close()

	payload, err := io.ReadAll(r.Body)
	if err != nil {
		log.Println("invalid payload", err)
		http.Error(w, "invalid payload", http.StatusBadRequest)
		return
	}
	if len(payload) == 0 {
		http.Error(w, "empty Payload", http.StatusBadRequest)
		return
	}
	//verify the Stripe Signature
	sig := r.Header.Get("Stripe-Signature")
	stripeWebhookSecret := os.Getenv("STRIPE_WEBHOOK_SECRET")
	if stripeWebhookSecret == "" {
		http.Error(w, "missing webhook secret", http.StatusInternalServerError)
		return
	}
	//construct the webhook event
	event, err := webhook.ConstructEventWithOptions(
		payload,
		sig,
		stripeWebhookSecret,
		webhook.ConstructEventOptions{
			IgnoreAPIVersionMismatch: true,
		},
	)
	if err != nil {
		log.Println(err)
		http.Error(w, "invalid signature", http.StatusBadRequest)
		return
	}
	//log.Printf("evt_id: %s", event.ID)

	//presist Event
	eventDetails := model.EventDetails{
		PspName: "stripe",
		EventID: event.ID,
		Payload: payload,
	}
	err = repo.StoreWebhookEvent(eventDetails)
	if err != nil {
		log.Println("failed to persist webhook:", err)
		http.Error(w, "failed to persist", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}
