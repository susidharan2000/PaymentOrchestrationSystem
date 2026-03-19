package reconciler

import (
	"encoding/json"
	"time"
)

type Payment struct {
	PaymentId string
	Status    string
	Amount    int64
	Currency  string
	PspName   string
	PspRefID  *string
	CreatedAt time.Time
}

type Refund struct {
	PaymentID   string
	RefundID    string
	Status      string
	Amount      int64
	Currency    string
	PspName     string
	PspRefundID *string
	CreatedAt   time.Time
}

type EventLogDetails struct {
	PspName      string
	PspEventID   string
	PspEventType string
	RawPayload   json.RawMessage
}
