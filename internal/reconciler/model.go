package reconciler

type Payment struct {
	PaymentId string
	Status    string
	Amount    int64
	Currency  string
	PspName   string
	PspRefID  *string
}
