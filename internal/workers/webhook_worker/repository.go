package webhookworker

import (
	"database/sql"
	"fmt"
)

type workerRepository interface {
	BeginTx() (*sql.Tx, error)
	ClaimEvents(limit int) ([]EventDetails, error)
	MarkProcessedTx(tx *sql.Tx, id int64) error
	MarkEventFailed(id int64) error
	//payment
	RecordPaymentSuccess(paymentDetails WebhookPaymentDetails, tx *sql.Tx) error
	MarkPaymentFailed(paymentDetails WebhookPaymentDetails, tx *sql.Tx) error
	//refund
	RecordRefundSuccess(pspRefundID string, tx *sql.Tx) error
	RecordRefundFailed(pspRefundID string, tx *sql.Tx) error
}

type repo struct {
	db *sql.DB
}

// constructure func
func NewWebhookWorkerRepository(db *sql.DB) workerRepository {
	return &repo{db: db}
}

func (r *repo) BeginTx() (*sql.Tx, error) {
	return r.db.Begin()
}

func (r *repo) MarkProcessedTx(tx *sql.Tx, id int64) error {
	res, err := tx.Exec(`
        UPDATE payment.webhook_events
        SET status = 'PROCESSED'
        WHERE id = $1
        AND status = 'PROCESSING'
    `, id)
	if err != nil {
		return err
	}

	rows, err := res.RowsAffected()
	if err != nil {
		return err
	}

	if rows == 0 {
		return fmt.Errorf("no rows updated for id=%d", id)
	}

	return nil
}

func (r *repo) MarkEventFailed(id int64) error {
	_, err := r.db.Exec(`
        UPDATE payment.webhook_events
        SET status = 'FAILED'
        WHERE id = $1
        AND status = 'PROCESSING'
        AND attempts >= 5
    `, id)
	return err
}

func (r *repo) ClaimEvents(limit int) ([]EventDetails, error) {
	rows, err := r.db.Query(`
        WITH rows AS (
            SELECT id
            FROM payment.webhook_events
            WHERE
                status IN ('PENDING', 'PROCESSING')
                AND next_retry_at <= now()
				AND attempts < 5
            ORDER BY next_retry_at ASC, id ASC
            FOR UPDATE SKIP LOCKED
            LIMIT $1
        )
        UPDATE payment.webhook_events w
        SET
            status = 'PROCESSING',
            attempts = attempts + 1,
            next_retry_at = now() + interval '2 minutes'
        FROM rows
        WHERE w.id = rows.id
        RETURNING w.id, w.payload;
    `, limit)

	if err != nil {
		return nil, err
	}
	defer rows.Close()

	events := make([]EventDetails, 0, limit)

	for rows.Next() {
		var e EventDetails
		if err := rows.Scan(&e.ID, &e.Payload); err != nil {
			return nil, err
		}
		events = append(events, e)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return events, nil
}

// payment success handler
func (r *repo) RecordPaymentSuccess(paymentDetails WebhookPaymentDetails, tx *sql.Tx) error {
	//insert Ledger
	query := `INSERT INTO payment.ledger_entries (entry_type, payment_id, amount, currency, psp_name, psp_ref_id) SELECT $2, pi.payment_id, $3, $4, $5, $1
        FROM payment.payment_intent pi
        WHERE pi.psp_ref_id = $1
        AND pi.status = 'PROCESSING'
        ON CONFLICT (psp_name, psp_ref_id, entry_type) DO NOTHING;
	`
	res, err := tx.Exec(query, paymentDetails.PiID, "PAYMENT", paymentDetails.Amount, paymentDetails.Currency, paymentDetails.PspName)
	if err != nil {
		return err
	}
	rows, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return fmt.Errorf("no ledger entry created for pi = %v", paymentDetails.PiID)
	}
	return nil
}

//payment failure handler

func (r *repo) MarkPaymentFailed(paymentDetails WebhookPaymentDetails, tx *sql.Tx) error {
	//update the payment_intnent
	res, err := tx.Exec(`UPDATE payment.payment_intent SET status = 'FAILED' WHERE psp_ref_id = $1 AND status = 'PROCESSING'`, paymentDetails.PiID)
	if err != nil {
		return err
	}
	rows, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return fmt.Errorf("no payment updated for pi = %v", paymentDetails.PiID)
	}
	return nil
}

// refund Success handler
func (r *repo) RecordRefundSuccess(pspRefundID string, tx *sql.Tx) error {
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
	ON CONFLICT (psp_name, psp_ref_id, entry_type) DO NOTHING;
	`
	res, err := tx.Exec(query, "REFUND", pspRefundID)
	if err != nil {
		return err
	}
	rows, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return fmt.Errorf("no ledger entry created for refund = %v", pspRefundID)
	}

	//update refund Record
	res, err = tx.Exec(`UPDATE payment.refund_record SET status = 'SUCCEEDED' WHERE psp_refund_id = $1 AND status = 'PROCESSING'`, pspRefundID)
	if err != nil {
		return err
	}
	rows, err = res.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return fmt.Errorf("no refund updated for refund = %v", pspRefundID)
	}
	return nil
}

// refund Failure handler
func (r *repo) RecordRefundFailed(pspRefundID string, tx *sql.Tx) error {
	//update refund Record
	res, err := tx.Exec(`UPDATE payment.refund_record SET status = 'FAILED' WHERE psp_refund_id = $1 AND status = 'PROCESSING'`, pspRefundID)
	if err != nil {
		return err
	}
	rows, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return fmt.Errorf("no refund updated for refund = %v", pspRefundID)
	}
	return nil
}
