package stripe

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"os"

	"github.com/stripe/stripe-go/v76/webhook"
)

func Webhook_ingestor(w http.ResponseWriter, r *http.Request) {
	payload, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "invalid payload", http.StatusBadRequest)
		return
	}
	sig := r.Header.Get("Stripe-Signature")
	stripeWebhookSecret := os.Getenv("STRIPE_WEBHOOK_SECRET")
	if stripeWebhookSecret == "" {
		http.Error(w, "missing webhook secret", http.StatusInternalServerError)
		return
	}
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
	fmt.Println(event.ID)
	fmt.Println("signature verified")
	w.WriteHeader(200)

}
