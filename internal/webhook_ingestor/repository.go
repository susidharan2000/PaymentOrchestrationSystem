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
	// Append in the psp_ledger entries
	// if _, err := tx.Exec(`INSERT INTO payment.ledger_entries (entry_type,payment_id,amount,currency,psp_name,psp_ref_id) VALUES ($1,null,$2,$3,$4,$5) ON CONFLICT (psp_name, psp_ref_id) DO NOTHING`, paymentStatus, paymentDetails.Amount, paymentDetails.Currency, paymentDetails.PspName, paymentDetails.PiID); err != nil {
	// 	return err
	// }
	query := `INSERT INTO payment.ledger_entries (entry_type, payment_id, amount, currency, psp_name, psp_ref_id) SELECT $2, pi.payment_id, $3, $4, $5, $1
        FROM payment.payment_intent pi
        WHERE pi.psp_ref_id = $1
        AND pi.status = 'PROCESSING'
        ON CONFLICT (psp_name, psp_ref_id) DO NOTHING;
	`
	if _, err := tx.Exec(query, paymentDetails.PiID, paymentStatus, paymentDetails.Amount, paymentDetails.Currency, paymentDetails.PspName); err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return err
	}
	isCommited = true
	return nil
}
