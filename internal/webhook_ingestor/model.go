package webhookingestor

type PaymemtIntentDetails struct {
	PaymentId string
	PiID      string
	Amount    int64
	Status    string
	Currency  string
	PspName   string
}
