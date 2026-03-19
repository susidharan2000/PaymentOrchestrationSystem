package webhookingestor

import (
	"database/sql"

	model "github.com/susidharan/payment-orchestration-system/internal/webhook_ingestor/model"
)

type WebhookRepository interface {
	//payment
	ProcessSuccessPaymentWebhook(paymentDetails model.WebhookPaymentDetails, eventDetails model.EventDetails) error
	ProcessFailedPaymentWebhook(paymentDetails model.WebhookPaymentDetails, eventDetails model.EventDetails) error
	//refund
	ProcessSuccessRefundWebhook(refundID string, eventDetails model.EventDetails) error
	ProcessFailedRefundWebhook(pspRefundID string, eventDetails model.EventDetails) error
}
type repo struct {
	db *sql.DB
}

func NewWebhookRepository(db *sql.DB) WebhookRepository {
	return &repo{db: db}
}

// payment success handler
func (r *repo) ProcessSuccessPaymentWebhook(paymentDetails model.WebhookPaymentDetails, eventDetails model.EventDetails) error {
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
	//insert webhook_event
	res, err := tx.Exec(`INSERT INTO payment.webhook_events (event_id,psp_name) VALUES ($1,$2) ON CONFLICT (event_id,psp_name) DO NOTHING ;`, eventDetails.EventID, eventDetails.PspName)
	if err != nil {
		return err
	}
	rows, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		if err := tx.Commit(); err != nil {
			return err
		}
		isCommited = true
		return nil
	}
	//insert Ledger
	query := `INSERT INTO payment.ledger_entries (entry_type, payment_id, amount, currency, psp_name, psp_ref_id) SELECT $2, pi.payment_id, $3, $4, $5, $1
        FROM payment.payment_intent pi
        WHERE pi.psp_ref_id = $1
        AND pi.status = 'PROCESSING'
        ON CONFLICT (psp_name, psp_ref_id) DO NOTHING;
	`
	_, err = tx.Exec(query, paymentDetails.PiID, "PAYMENT", paymentDetails.Amount, paymentDetails.Currency, paymentDetails.PspName)
	if err != nil {
		return err
	}

	if err := tx.Commit(); err != nil {
		return err
	}
	isCommited = true
	return nil
}

//payment failure handler

func (r *repo) ProcessFailedPaymentWebhook(paymentDetails model.WebhookPaymentDetails, eventDetails model.EventDetails) error {
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
	//insert webhook_event
	res, err := tx.Exec(`INSERT INTO payment.webhook_events (event_id,psp_name) VALUES ($1,$2) ON CONFLICT (event_id,psp_name) DO NOTHING ;`, eventDetails.EventID, eventDetails.PspName)
	if err != nil {
		return err
	}
	rows, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		if err := tx.Commit(); err != nil {
			return err
		}
		isCommited = true
		return nil
	}
	//update the payment_intnent
	_, err = tx.Exec(`UPDATE payment.payment_intent SET status = 'FAILED' WHERE psp_ref_id = $1 AND status = 'PROCESSING'`, paymentDetails.PiID)
	if err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return err
	}
	isCommited = true
	return nil
}

// refund Success handler
func (r *repo) ProcessSuccessRefundWebhook(pspRefundID string, eventDetails model.EventDetails) error {

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

	//insert webhook_event
	res, err := tx.Exec(`INSERT INTO payment.webhook_events (event_id,psp_name) VALUES ($1,$2) ON CONFLICT (event_id,psp_name) DO NOTHING ;`, eventDetails.EventID, eventDetails.PspName)
	if err != nil {
		return err
	}
	rows, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		if err := tx.Commit(); err != nil {
			return err
		}
		isCommited = true
		return nil
	}

	//Append the ledger
	query := `INSERT INTO payment.ledger_entries 
	(entry_type, payment_id, amount, currency, psp_name, psp_ref_id, refund_id)
	SELECT 
		$1,
		r.payment_id,
		r.amount,
		r.currency,
		r.psp_name,
		r.psp_refund_id,
		r.refund_entry_id
	FROM payment.refund_record r
	WHERE r.psp_refund_id = $2
	AND r.status = 'PROCESSING'
	AND r.psp_refund_id IS NOT NULL
	ON CONFLICT (psp_name, psp_ref_id) DO NOTHING;
	`
	if _, err := tx.Exec(query, "REFUND", pspRefundID); err != nil {
		return err
	}
	//indert refund Record
	_, err = tx.Exec(`UPDATE payment.refund_record SET status = 'SUCCEEDED' WHERE psp_refund_id = $1 AND status = 'PROCESSING'`, pspRefundID)
	if err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return err
	}
	isCommited = true
	return nil
}

// refund Failure handler
func (r *repo) ProcessFailedRefundWebhook(pspRefundID string, eventDetails model.EventDetails) error {
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

	//insert webhook_event
	res, err := tx.Exec(`INSERT INTO payment.webhook_events (event_id,psp_name) VALUES ($1,$2) ON CONFLICT (event_id,psp_name) DO NOTHING ;`, eventDetails.EventID, eventDetails.PspName)
	if err != nil {
		return err
	}
	rows, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		if err := tx.Commit(); err != nil {
			return err
		}
		isCommited = true
		return nil
	}
	//insert refund Record
	_, err = tx.Exec(`UPDATE payment.refund_record SET status = 'FAILED' WHERE psp_refund_id = $1 AND status = 'PROCESSING'`, pspRefundID)
	if err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return err
	}
	isCommited = true
	return nil
}
