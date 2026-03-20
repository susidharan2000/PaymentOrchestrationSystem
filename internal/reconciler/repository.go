package reconciler

import (
	"database/sql"
)

type ReconcilerRepository interface {
	//payment
	ClaimUnresolvedPayments(limit int) ([]Payment, error)
	RecordPaymentSuccess(payment Payment) error
	MarkPaymentFailed(paymentDetails Payment) error

	//refund
	ClaimUnresolvedRefunds(limit int) ([]Refund, error)
	RefundSuccessEntry(refundPayment Refund) error
	MarkRefundFailed(refundPayment Refund) error
}

type repo struct {
	db *sql.DB
}

func NewReconcilerRepository(db *sql.DB) ReconcilerRepository {
	return &repo{db: db}
}

// payment
func (r *repo) ClaimUnresolvedPayments(limit int) ([]Payment, error) {
	//implement Lease Claim Pattern to claim payment
	// - because it avoids the multiple unnessary psp calls
	// - safe fop multiple reconciler instance
	payments := make([]Payment, 0, limit)
	query := `WITH rows AS(
		SELECT payment_id
		FROM payment.payment_intent
		WHERE status IN ('PROCESSING','FAILED') AND 
		psp_ref_id IS NOT NULL AND
		(
			next_reconcile_at <= now()
		)
		ORDER BY next_reconcile_at ASC, payment_id ASC
		FOR UPDATE SKIP LOCKED
		LIMIT $1
		)
		UPDATE payment.payment_intent p
		SET reconcile_attempts = reconcile_attempts+1,
		updated_at = now(),
		next_reconcile_at = 
		    CASE
                WHEN now() - p.created_at < interval '5 minutes'
                    THEN now() + interval '30 seconds'

                WHEN now() - p.created_at < interval '30 minutes'
                    THEN now() + interval '2 minutes'

                WHEN now() - p.created_at < interval '2 hours'
                    THEN now() + interval '10 minutes'
					
                ELSE now() + interval '1 hour'
            END
		FROM rows
		WHERE p.payment_id = rows.payment_id
		RETURNING p.payment_id, p.status, p.amount, p.currency, p.psp_name, p.psp_ref_id, p.created_at;
		`
	rows, err := r.db.Query(query, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var payment Payment
		err := rows.Scan(&payment.PaymentId, &payment.Status, &payment.Amount, &payment.Currency, &payment.PspName, &payment.PspRefID, &payment.CreatedAt)
		if err != nil {
			return nil, err
		}
		payments = append(payments, payment)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return payments, nil
}

// payment success handler
func (r *repo) RecordPaymentSuccess(paymentDetails Payment) error {
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
	query := `INSERT INTO payment.ledger_entries (entry_type, payment_id, amount, currency, psp_name, psp_ref_id) SELECT 'PAYMENT', pi.payment_id, $2, $3, $4, $1
        FROM payment.payment_intent pi
        WHERE pi.psp_ref_id = $1
        ON CONFLICT (psp_name, psp_ref_id) DO NOTHING;
	`
	if _, err := tx.Exec(query, paymentDetails.PspRefID, paymentDetails.Amount, paymentDetails.Currency, paymentDetails.PspName); err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return err
	}
	isCommited = true
	return nil
}

// payment Failure handler
func (r *repo) MarkPaymentFailed(paymentDetails Payment) error {
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
	//update the payment_intnent
	_, err = tx.Exec(`UPDATE payment.payment_intent SET status = 'FAILED' WHERE psp_ref_id = $1 AND status = 'PROCESSING'`, paymentDetails.PspRefID)
	if err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return err
	}
	isCommited = true
	return nil
}

// refund
func (r *repo) ClaimUnresolvedRefunds(limit int) ([]Refund, error) {
	//implement Lease Claim Pattern to claim payment
	// - because it avoids the multiple unnessary psp calls
	// - safe fop multiple reconciler instance
	refunds := make([]Refund, 0, limit)
	query := `WITH rows AS(
		SELECT refund_entry_id
		FROM payment.refund_record
		WHERE status IN ('PROCESSING','FAILED')
		AND psp_refund_id IS NOT NULL 
		AND next_reconcile_at <= now()
		ORDER BY next_reconcile_at ASC, refund_entry_id ASC
		FOR UPDATE SKIP LOCKED
		LIMIT $1
		)
		UPDATE payment.refund_record r
		SET reconcile_attempts = reconcile_attempts+1,
		updated_at = now(),
		next_reconcile_at = 
		    CASE
                WHEN now() - r.created_at < interval '5 minutes'
                    THEN now() + interval '30 seconds'

                WHEN now() - r.created_at < interval '30 minutes'
                    THEN now() + interval '2 minutes'

                WHEN now() - r.created_at < interval '2 hours'
                    THEN now() + interval '10 minutes'
					
                ELSE now() + interval '1 hour'
            END
		FROM rows
		WHERE r.refund_entry_id = rows.refund_entry_id
		RETURNING r.refund_entry_id, r.payment_id, r.status, r.amount, r.currency, r.psp_name, r.psp_refund_id, r.created_at;
		`
	rows, err := r.db.Query(query, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var refund Refund
		err := rows.Scan(&refund.RefundID,
			&refund.PaymentID,
			&refund.Status,
			&refund.Amount,
			&refund.Currency,
			&refund.PspName,
			&refund.PspRefundID,
			&refund.CreatedAt)
		if err != nil {
			return nil, err
		}
		refunds = append(refunds, refund)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return refunds, nil
}

// Refund Success entry
func (r *repo) RefundSuccessEntry(refundPayment Refund) error {
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
	query := `INSERT INTO payment.ledger_entries 
	(entry_type, payment_id, amount, currency, psp_name, psp_ref_id, refund_id)
	SELECT 
		'REFUND',
		r.payment_id,
		r.amount,
		r.currency,
		r.psp_name,
		r.psp_refund_id,
		r.refund_entry_id
	FROM payment.refund_record r
	WHERE r.refund_entry_id = $1
	AND r.psp_refund_id IS NOT NULL
	ON CONFLICT (psp_name, psp_ref_id) DO NOTHING;
	`
	_, err = tx.Exec(query, refundPayment.RefundID)
	if err != nil {
		return err
	}

	if _, err := tx.Exec(`UPDATE payment.refund_record SET status = 'SUCCEEDED' WHERE refund_entry_id = $1 AND status = 'PROCESSING';
	`, refundPayment.RefundID); err != nil {
		return err
	}

	if err := tx.Commit(); err != nil {
		return err
	}
	isCommited = true
	return nil
}

// refund Failure entry
func (r *repo) MarkRefundFailed(refundPayment Refund) error {
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
	if _, err := tx.Exec(`UPDATE payment.refund_record SET status = 'FAILED' WHERE refund_entry_id = $1 AND status = 'PROCESSING';
	`, refundPayment.RefundID); err != nil {
		return err
	}

	if err := tx.Commit(); err != nil {
		return err
	}
	isCommited = true
	return nil
}
