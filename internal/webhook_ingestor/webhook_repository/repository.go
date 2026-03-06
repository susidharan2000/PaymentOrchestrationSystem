package webhookingestor

import (
	"database/sql"

	model "github.com/susidharan/payment-orchestration-system/internal/webhook_ingestor/model"
)

type WebhookRepository interface {
	ProcessWebhook(paymentDetails model.WebhookPaymentDetails, paymentStatus string) error
}
type repo struct {
	db *sql.DB
}

func NewWebhookRepository(db *sql.DB) WebhookRepository {
	return &repo{db: db}
}

func (r *repo) ProcessWebhook(paymentDetails model.WebhookPaymentDetails, paymentStatus string) error {
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
