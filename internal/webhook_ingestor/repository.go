package webhookingestor

import (
	"database/sql"
)

type WebhookRepository interface {
	AppendLedger(paymentDetails WebhookPaymentDetails, paymentStatus string) error
}
type repo struct {
	db *sql.DB
}

func NewWebhookRepository(db *sql.DB) WebhookRepository {
	return &repo{db: db}
}

func (r *repo) AppendLedger(paymentDetails WebhookPaymentDetails, paymentStatus string) error {

	_, err := r.db.Exec(`INSERT INTO payment.ledger_entries (entry_type,payment_id,amount,currency,psp_name,psp_ref_id) VALUES ($1,null,$2,$3,$4,$5) ON CONFLICT (psp_name, psp_ref_id) DO NOTHING`, paymentStatus, paymentDetails.Amount, paymentDetails.Currency, paymentDetails.PspName, paymentDetails.PiID)
	if err != nil {
		return err
	}
	return nil
}
