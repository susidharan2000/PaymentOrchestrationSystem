package webhookingestor

type WebhookPaymentDetails struct {
	PiID      string
	PaymentId *string
	Amount    int64
	Status    string
	Currency  string
	PspName   string
}
