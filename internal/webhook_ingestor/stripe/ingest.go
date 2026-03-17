package stripe

import (
	"encoding/json"
	"io"
	"log"
	"net/http"
	"os"

	"github.com/stripe/stripe-go/v78"
	"github.com/stripe/stripe-go/v78/webhook"
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
	log.Printf("evt_id: %s", event.ID)
	//update the ledger
	switch event.Type {
	case "payment_intent.succeeded", "payment_intent.payment_failed":
		// extract the Payment Intent Details from the event
		var pi stripe.PaymentIntent
		if err = json.Unmarshal(event.Data.Raw, &pi); err != nil {
			http.Error(w, "failed to parse event", http.StatusInternalServerError)
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
		switch event.Type {
		case "payment_intent.succeeded":
			if err := updatePaymentResult(paymentDetails, "CAPTURED", repo); err != nil {
				http.Error(w, "ledger write failed", http.StatusInternalServerError)
				return
			}
		case "payment_intent.payment_failed":
			if err := updatePaymentResult(paymentDetails, "FAILED", repo); err != nil {
				http.Error(w, "ledger write failed", http.StatusInternalServerError)
				return
			}
		}
	case "refund.updated":
		var r stripe.Refund
		if err = json.Unmarshal(event.Data.Raw, &r); err != nil {
			http.Error(w, "failed to parse event", http.StatusInternalServerError)
			return
		}
		log.Printf("Refund Status: %s", r.Status)
		switch r.Status {
		case stripe.RefundStatusSucceeded:
			log.Printf("pspRefund_id: %s", r.ID)
			if err := updateRefundStatus(r.ID, "REFUND", repo); err != nil {
				http.Error(w, "ledger write failed", http.StatusInternalServerError)
				return
			}
		case stripe.RefundStatusFailed:
			log.Printf("pspRefund_id: %s", r.ID)
			if err := updateRefundStatus(r.ID, "FAILED", repo); err != nil {
				http.Error(w, "ledger write failed", http.StatusInternalServerError)
				return
			}
		default:

		}

	default:

	}
	w.WriteHeader(http.StatusOK)
}

// handle payment_intent Webhook
func updatePaymentResult(paymentDetails model.WebhookPaymentDetails, paymentStatus string, repo WebhookRepo.WebhookRepository) error {
	//process the wehhook
	if err := repo.ProcessPaymentWebhook(paymentDetails, paymentStatus); err != nil {
		log.Println(err)
		return err
	}
	log.Println("Webhook Process Completed")
	return nil
}

// handle refund Webhook
func updateRefundStatus(refundID string, paymentStatus string, repo WebhookRepo.WebhookRepository) error {
	//process the refund Webhook
	if err := repo.ProcessRefundWebhook(refundID, paymentStatus); err != nil {
		log.Println(err)
		return err
	}
	log.Println("Webhook Process Completed")
	return nil
}
