package intent

// request from client
type CreatePaymentRequest struct {
	Amount         int64  `json:"amount"`
	Currency       string `json:"currency"`
	PspName        string `json:"psp_name"`
	IdempotencyKey string `json:"idempotency_key"`
}

// responce to client
type PaymentDetails struct {
	PaymentId string `json:"payment_id"`
	Amount    int64  `json:"amount"`
	Currency  string `json:"currency"`
	Status    string `json:"status"`
	PspName   string `json:"psp_name"`
}
