package stripe

import (
	"encoding/json"
	"io"
	"log"
	"net/http"
	"os"

	"github.com/stripe/stripe-go/v76"
	"github.com/stripe/stripe-go/v76/webhook"
	Webhook "github.com/susidharan/payment-orchestration-system/internal/webhook_ingestor"
)

func Webhook_ingestor(w http.ResponseWriter, r *http.Request, repo Webhook.WebhookRepository) {
	payload, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "invalid payload", http.StatusBadRequest)
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
	// extract the Payment Intent Details from the event
	var pi stripe.PaymentIntent
	raw, err := json.Marshal(event.Data.Object)
	if err != nil {
		log.Println(err)
		return
	}
	if err = json.Unmarshal(raw, &pi); err != nil {
		return
	}
	piID := pi.ID
	paymentID := pi.Metadata["payment_id"]
	//updat the ledger
	switch event.Type {
	case "payment_intent.succeeded":
		updateWebhookResult(piID, paymentID, "CAPTURED", repo)
	case "payment_intent.payment_failed":
		updateWebhookResult(piID, paymentID, "FAILED", repo)
	default:
		w.WriteHeader(http.StatusOK)
	}
	w.WriteHeader(http.StatusOK)
}

func updateWebhookResult(piID string, paymentID string, paymentStatus string, repo Webhook.WebhookRepository) {
	// Store  the psp event (log) - this should be first because id the crash before appending the "Failed State"
	// update the ledger entry
	repo.AppendLedger(piID, paymentID, paymentStatus)
	// if the ledger is success full then update the payment_intent
}
