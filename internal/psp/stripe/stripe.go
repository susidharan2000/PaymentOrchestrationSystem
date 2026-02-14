package stripe

import (
	"os"

	"github.com/stripe/stripe-go/v78"
	"github.com/stripe/stripe-go/v78/paymentintent"
	"github.com/susidharan/payment-orchestration-system/internal/domain"
)

func Init() {
	stripe.Key = os.Getenv("STRIPE_SECRET_KEY")
}

func CreatePaymentIntent(paymentDetails domain.PaymentParams) (string, error) {
	params := &stripe.PaymentIntentParams{
		Amount:   stripe.Int64(paymentDetails.Amount * 100),
		Currency: stripe.String(paymentDetails.Currency),
		Metadata: map[string]string{
			"payment_id": paymentDetails.PaymentId,
		},
		//for testing only
		AutomaticPaymentMethods: &stripe.PaymentIntentAutomaticPaymentMethodsParams{
			Enabled:        stripe.Bool(true),
			AllowRedirects: stripe.String("never"),
		},
		Confirm:       stripe.Bool(true),
		PaymentMethod: stripe.String("pm_card_chargeDeclined"), // for fail payment
		//PaymentMethod: stripe.String("pm_card_visa"), // for the success payment

	}
	params.SetIdempotencyKey(paymentDetails.PaymentId)
	pi, err := paymentintent.New(params)
	if err != nil {
		return "", err
	}
	return pi.ID, nil
}
