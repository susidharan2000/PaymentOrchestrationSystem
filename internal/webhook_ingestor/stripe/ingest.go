package stripe

import (
	"encoding/json"
	"io"
	"log"
	"net/http"
	"os"

	"github.com/stripe/stripe-go/v76"
	"github.com/stripe/stripe-go/v76/webhook"
	model "github.com/susidharan/payment-orchestration-system/internal/webhook_ingestor/model"
	WebhookRepo "github.com/susidharan/payment-orchestration-system/internal/webhook_ingestor/webhook_repository"
)

func Webhook_ingestor(w http.ResponseWriter, r *http.Request, repo WebhookRepo.WebhookRepository) {
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
	if string(event.Type) != "payment_intent.succeeded" && string(event.Type) != "payment_intent.payment_failed" {
		return
	}
	log.Printf("evt_id: %s", event.ID)
	// extract the Payment Intent Details from the event
	var pi stripe.PaymentIntent
	raw, err := json.Marshal(event.Data.Object)
	if err != nil {
		http.Error(w, "missing webhook secret", http.StatusInternalServerError)
		return
	}
	if err = json.Unmarshal(raw, &pi); err != nil {
		return
	}
	piID := pi.ID
	currency := pi.Currency
	amount := pi.Amount

	var paymentDetails model.WebhookPaymentDetails

	paymentDetails.PiID = piID
	paymentDetails.Currency = string(currency)
	paymentDetails.Amount = amount
	paymentDetails.PspName = "stripe"

	log.Printf("PiID: %s", piID)
	//update the ledger
	switch event.Type {
	case "payment_intent.succeeded":
		if err := updateWebhookResult(paymentDetails, "CAPTURED", repo); err != nil {
			http.Error(w, "ledger write failed", http.StatusInternalServerError)
			return
		}
	case "payment_intent.payment_failed":
		if err := updateWebhookResult(paymentDetails, "FAILED", repo); err != nil {
			http.Error(w, "ledger write failed", http.StatusInternalServerError)
			return
		}
	default:

	}
	w.WriteHeader(http.StatusOK)
}

func updateWebhookResult(paymentDetails model.WebhookPaymentDetails, paymentStatus string, repo WebhookRepo.WebhookRepository) error {
	//process the wehhook
	if err := repo.ProcessWebhook(paymentDetails, paymentStatus); err != nil {
		log.Println(err)
		return err
	}
	log.Println("Webhook Process Completed")
	return nil
}
