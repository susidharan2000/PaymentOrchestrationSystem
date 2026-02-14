package webhookingestor

import (
	"database/sql"
)

type WebhookRepository interface {
	ProcessWebhook(paymentDetails WebhookPaymentDetails, ventLogDetails EventLogDetails, paymentStatus string) error
}
type repo struct {
	db *sql.DB
}

func NewWebhookRepository(db *sql.DB) WebhookRepository {
	return &repo{db: db}
}

func (r *repo) ProcessWebhook(paymentDetails WebhookPaymentDetails, eventLogDetails EventLogDetails, paymentStatus string) error {
	isCommited := false
	tx, err := r.db.Begin()
	if err != nil {
		return err
	}
	defer func() {
		if !isCommited {
			tx.Rollback()
		}
	}()
	// store the raw psp event
	if _, err := tx.Exec(`INSERT INTO payment.psp_events (psp_name, psp_event_id, psp_event_type, raw_payload) VALUES ($1, $2, $3, $4) ON CONFLICT (psp_name, psp_event_id) DO NOTHING;`,
		eventLogDetails.PspName,
		eventLogDetails.PspEventID,
		eventLogDetails.PspEventType,
		eventLogDetails.RawPayload,
	); err != nil {
		return err
	}
	// Append in the psp_ledger entries
	if _, err := tx.Exec(`INSERT INTO payment.ledger_entries (entry_type,payment_id,amount,currency,psp_name,psp_ref_id) VALUES ($1,null,$2,$3,$4,$5) ON CONFLICT (psp_name, psp_ref_id) DO NOTHING`, paymentStatus, paymentDetails.Amount, paymentDetails.Currency, paymentDetails.PspName, paymentDetails.PiID); err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return err
	}
	isCommited = true
	return nil
}
