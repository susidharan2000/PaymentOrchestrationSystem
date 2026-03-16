package worker

import (
	"database/sql"
	"fmt"
)

type workerRepository interface {
	ClaimRefundablePayment() (paymentDetails RefundDetails, err error)
	MarkProcessing(PaymentId string, pspReferenceID string) error
}

type repo struct {
	db *sql.DB
}

// constructure func
func NewReundWorkerRepository(db *sql.DB) workerRepository {
	return &repo{db: db}
}
func (r *repo) ClaimRefundablePayment() (RefundDetails, error) {
	var refundDetails RefundDetails
	tx, err := r.db.Begin()
	if err != nil {
		return RefundDetails{}, err
	}
	committed := false
	defer func() {
		if !committed {
			tx.Rollback()
		}
	}()
	err = tx.QueryRow(`WITH row AS (
		SELECT refund_entry_id
		FROM payment.refund_record
		WHERE status = 'PENDING'
		AND psp_refund_id IS NULL 
		AND (next_retry_at <= now())
		AND retry_count < max_retry
		FOR UPDATE SKIP LOCKED
		LIMIT 1
	)
	UPDATE payment.refund_record
	SET next_retry_at = now() + (interval '30 seconds' * power(2, retry_count)) + (interval '5 seconds' * random()),
	retry_count = retry_count + 1
	FROM row
	WHERE payment.refund_record.refund_entry_id = row.refund_entry_id
	RETURNING payment.refund_record.refund_entry_id, payment.refund_record.amount, payment.refund_record.psp_name, payment.refund_record.psp_payment_ref_id;
	`).Scan(&refundDetails.refundID, &refundDetails.amount, &refundDetails.PspName, &refundDetails.pspReferenceID)
	if err == sql.ErrNoRows {
		return RefundDetails{}, nil
	}
	if err != nil {
		return RefundDetails{}, err
	}
	if err := tx.Commit(); err != nil {
		return RefundDetails{}, err
	}
	committed = true
	return refundDetails, nil
}

func (r *repo) MarkProcessing(refundID string, pspRefundID string) error {
	result, err := r.db.Exec(`UPDATE payment.refund_record
	SET status = 'PROCESSING', psp_refund_id = $1
	WHERE refund_entry_id = $2
	AND status = 'PENDING'
	AND psp_refund_id IS NULL
	AND retry_count < max_retry;
	`, pspRefundID, refundID)
	if err != nil {
		return err
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return fmt.Errorf("refund %s could not be marked PROCESSING", refundID)
	}
	return nil
}
