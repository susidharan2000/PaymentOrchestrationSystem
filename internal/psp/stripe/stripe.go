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
	}
	pi, err := paymentintent.New(params)
	if err != nil {
		return "", err
	}
	return pi.ID, nil
}
