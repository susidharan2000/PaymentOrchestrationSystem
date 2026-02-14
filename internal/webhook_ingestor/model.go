package webhookingestor

import "encoding/json"

type WebhookPaymentDetails struct {
	PiID      string
	PaymentId *string
	Amount    int64
	Status    string
	Currency  string
	PspName   string
}

type EventLogDetails struct {
	PspName      string
	PspEventID   string
	PspEventType string
	RawPayload   json.RawMessage
}
