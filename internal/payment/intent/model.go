package intent

// request from client
type CreatePaymentRequest struct {
	Amount         int64  `json:"amount"`
	Currency       string `json:"currency"`
	PspName        string `json:"psp_name"`
	IdempotencyKey string `json:"idempotency_key"`
}

// responce to client (create payment)
type PaymentDetails struct {
	PaymentId      string `json:"payment_id"`
	Amount         int64  `json:"amount"`
	Currency       string `json:"currency"`
	Status         string `json:"status"`
	PspName        string `json:"psp_name"`
	ClientSecret   string `json:"client_secret"`
	Publishablekey string `json:"publishable_key"`
}

// for hashing the request
type PaymentFingerprint struct {
	Amount   int64
	Currency string
	PSPName  string
}

// responce ro client (get payment by ID)
type PaymentDetailsByID struct {
	Amount   int64  `json:"amount"`
	Currency string `json:"currency"`
	Status   string `json:"status"`
	PspName  string `json:"psp_name"`
}
