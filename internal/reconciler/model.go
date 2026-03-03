package reconciler

import "encoding/json"

type Payment struct {
	PaymentId string
	Status    string
	Amount    int64
	Currency  string
	PspName   string
	PspRefID  *string
}

type EventLogDetails struct {
	PspName      string
	PspEventID   string
	PspEventType string
	RawPayload   json.RawMessage
}
