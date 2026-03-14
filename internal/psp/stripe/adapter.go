package stripe

import (
	"os"

	"github.com/stripe/stripe-go/v78"
	"github.com/stripe/stripe-go/v78/paymentintent"
	"github.com/susidharan/payment-orchestration-system/internal/domain"
)

type Adapter struct{}

func NewAdapter() *Adapter {
	stripe.Key = os.Getenv("STRIPE_SECRET_KEY")
	return &Adapter{}
}

func (a *Adapter) CreatePaymentIntent(paymentDetails domain.PaymentParams) (string, string, error) {
	// Automatic Comformation
	// params := &stripe.PaymentIntentParams{
	// 	Amount:   stripe.Int64(paymentDetails.Amount * 100),
	// 	Currency: stripe.String(paymentDetails.Currency),
	// 	Metadata: map[string]string{
	// 		"payment_id": paymentDetails.PaymentId,
	// 	},
	// 	//for testing only
	// 	AutomaticPaymentMethods: &stripe.PaymentIntentAutomaticPaymentMethodsParams{
	// 		Enabled:        stripe.Bool(true),
	// 		AllowRedirects: stripe.String("never"),
	// 	},
	// 	Confirm: stripe.Bool(true),
	// 	//PaymentMethod: stripe.String("pm_card_chargeDeclined"), // for fail payment
	// 	PaymentMethod: stripe.String("pm_card_visa"), // for the success payment

	// }

	// Manual Conformation
	params := &stripe.PaymentIntentParams{
		Amount:   stripe.Int64(paymentDetails.Amount),
		Currency: stripe.String(paymentDetails.Currency),
		Metadata: map[string]string{
			"payment_id": paymentDetails.PaymentId,
		},
		AutomaticPaymentMethods: &stripe.PaymentIntentAutomaticPaymentMethodsParams{
			Enabled: stripe.Bool(true),
		},
		Confirm: stripe.Bool(false),
	}

	params.SetIdempotencyKey(paymentDetails.PaymentId)
	pi, err := paymentintent.New(params)
	if err != nil {
		if stripeErr, ok := err.(*stripe.Error); ok {
			if stripeErr.PaymentIntent != nil {
				return stripeErr.PaymentIntent.ID, "", err
			}
		}
		return "", "", err
	}
	return pi.ID, pi.ClientSecret, nil
}

func (a *Adapter) GetPaymentIntent(pspRefID string) (domain.PspIntent, bool, error) {

	pi, err := paymentintent.Get(pspRefID, nil)
	var retryable bool = false
	if err != nil {
		if stripeErr, ok := err.(*stripe.Error); ok {
			if stripeErr.HTTPStatusCode == 429 || stripeErr.HTTPStatusCode >= 500 {
				retryable = true
			}
		}
		return domain.PspIntent{}, retryable, err
	}

	var response domain.PspIntent
	response.ClientSecret = pi.ClientSecret
	switch pi.Status {

	case stripe.PaymentIntentStatusSucceeded:
		response.Status = domain.StatusSucceeded

	case stripe.PaymentIntentStatusCanceled:
		response.Status = domain.StatusFailed

	default:
		response.Status = domain.StatusProcessing
	}

	return response, false, nil
}
