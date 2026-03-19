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
		log.Println("invalid payload", err)
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
	eventDetails := model.EventDetails{
		PspName: "STRIPE",
		EventID: event.ID,
	}
	//update the ledger
	switch event.Type {
	case "payment_intent.succeeded", "payment_intent.payment_failed":
		// extract the Payment Intent Details from the event
		var pi stripe.PaymentIntent
		if err = json.Unmarshal(event.Data.Raw, &pi); err != nil {
			log.Println("failed to parse event", err)
			w.WriteHeader(http.StatusOK)
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
			//append only in ledger
			if err := repo.ProcessSuccessPaymentWebhook(paymentDetails, eventDetails); err != nil {
				log.Println("processing failed:", err)
				w.WriteHeader(http.StatusOK)
				return
			}
			log.Println("Payment Webhook Process Completed")
		case "payment_intent.payment_failed":
			//update in payment_intent
			if err := repo.ProcessFailedPaymentWebhook(paymentDetails, eventDetails); err != nil {
				log.Println("processing failed:", err)
				w.WriteHeader(http.StatusOK)
				return
			}
			log.Println("Payment Webhook Process Completed")
		}
	case "refund.updated":
		var r stripe.Refund
		if err = json.Unmarshal(event.Data.Raw, &r); err != nil {
			log.Println("failed to parse event", err)
			w.WriteHeader(http.StatusOK)
			return
		}
		log.Printf("Refund Status: %s", r.Status)
		switch r.Status {
		case stripe.RefundStatusSucceeded:
			//append only in ledger
			log.Printf("pspRefund_id: %s", r.ID)
			if err := repo.ProcessSuccessRefundWebhook(r.ID, eventDetails); err != nil {
				log.Println("processing failed:", err)
				w.WriteHeader(http.StatusOK)
				return
			}
			log.Println("Refund Webhook Process Completed")
		case stripe.RefundStatusFailed:
			//update in refund_record
			log.Printf("pspRefund_id: %s", r.ID)
			if err := repo.ProcessFailedRefundWebhook(r.ID, eventDetails); err != nil {
				log.Println("processing failed:", err)
				w.WriteHeader(http.StatusOK)
				return
			}
			log.Println("Refund Webhook Process Completed")
		default:
			log.Printf("Unhandled refund status: %s", r.Status)
		}
	default:
		log.Printf("Unhandled event type: %s", event.Type)
	}
	w.WriteHeader(http.StatusOK)
}
