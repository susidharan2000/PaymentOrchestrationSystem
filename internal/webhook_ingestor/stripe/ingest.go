package stripe

import (
	"database/sql"
	"encoding/json"
	"errors"
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
	log.Printf("PiID: %s , PaymentId: %s", piID, paymentID)
	//updat the ledger
	switch event.Type {
	case "payment_intent.succeeded":
		updateWebhookResult(piID, paymentID, "CAPTURED", repo)
	case "payment_intent.payment_failed":
		updateWebhookResult(piID, paymentID, "FAILED", repo)
	default:

	}
	w.WriteHeader(http.StatusOK)
}

func updateWebhookResult(piID string, paymentID string, paymentStatus string, repo Webhook.WebhookRepository) {
	log.Print("Updating Webhook")
	log.Printf("PiID: %s , PaymentId: %s", piID, paymentID)
	// query the payment_intern table and get all the data for the ledger
	paymentDetails, err := repo.GetPaymentDetails(paymentID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return
		}
		log.Println("unexpected error:", err)
		return
	}
	log.Print("paymentDetails retrived")
	paymentDetails.PiID = piID
	paymentDetails.PaymentId = paymentID
	paymentDetails.Status = paymentStatus
	log.Print("got the payment Details")
	// Store  the psp event (log) - this should be first because id the crash before appending the "Failed State"
	// update the ledger entry
	if err := repo.AppendLedger(paymentDetails); err != nil {
		log.Panicln(err)
		return
	}
	// if the ledger is success full then update the payment_intent
}
