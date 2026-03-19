package reconciler

import (
	"database/sql"
	"fmt"
)

type ReconcilerRepository interface {
	ClaimUnresolvedPayments(limit int) ([]Payment, error)
	ClaimUnresolvedRefunds(limit int) ([]Refund, error)
	AppendLedgerEntry(payment Payment, paymentStatus string) error
	RefundSuccessEntry(refundPayment Refund, refundStatus string) error
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
		WHERE status IN ('PROCESSING') AND 
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

//refund

func (r *repo) ClaimUnresolvedRefunds(limit int) ([]Refund, error) {
	//implement Lease Claim Pattern to claim payment
	// - because it avoids the multiple unnessary psp calls
	// - safe fop multiple reconciler instance
	refunds := make([]Refund, 0, limit)
	query := `WITH rows AS(
		SELECT refund_entry_id
		FROM payment.refund_record
		WHERE status = 'PROCESSING'
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

// payment
func (r *repo) AppendLedgerEntry(paymentDetails Payment, paymentStatus string) error {
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
	query := `INSERT INTO payment.ledger_entries (entry_type, payment_id, amount, currency, psp_name, psp_ref_id) SELECT $2, pi.payment_id, $3, $4, $5, $1
        FROM payment.payment_intent pi
        WHERE pi.psp_ref_id = $1
        AND pi.status = 'PROCESSING'
        ON CONFLICT (psp_name, psp_ref_id) DO NOTHING;
	`
	if _, err := tx.Exec(query, paymentDetails.PspRefID, paymentStatus, paymentDetails.Amount, paymentDetails.Currency, paymentDetails.PspName); err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return err
	}
	isCommited = true
	return nil
}

// Refund Success entry
func (r *repo) RefundSuccessEntry(refundPayment Refund, refundStatus string) error {
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
		$1,
		r.payment_id,
		r.amount,
		r.currency,
		r.psp_name,
		r.psp_refund_id,
		$2
	FROM payment.refund_record r
	WHERE r.refund_entry_id = $2
	AND r.status = 'PROCESSING'
	AND r.psp_refund_id IS NOT NULL
	ON CONFLICT (psp_name, psp_ref_id) DO NOTHING;
	`
	res, err := tx.Exec(query, refundStatus, refundPayment.RefundID)
	if err != nil {
		return err
	}
	rows, _ := res.RowsAffected()
	if rows == 0 {
		//ambiguous case
		var status string
		err = tx.QueryRow(`
		SELECT status FROM payment.refund_record
		WHERE refund_entry_id = $1
	`, refundPayment.RefundID).Scan(&status)

		if err != nil {
			return err
		}
		if status == "SUCCEEDED" || status == "FAILED" {
			return tx.Commit()
		}
		return fmt.Errorf("invalid state: ledger not inserted and still processing")
	}
	// update the state in the refund Record
	var refundRecordStatus string
	if refundStatus == "REFUND" {
		refundRecordStatus = "SUCCEEDED"
	} else {
		refundRecordStatus = "FAILED"
	}
	if _, err := tx.Exec(`UPDATE payment.refund_record SET status = $1 WHERE refund_entry_id = $2 AND status = 'PROCESSING';
	`, refundRecordStatus, refundPayment.RefundID); err != nil {
		return err
	}

	if err := tx.Commit(); err != nil {
		return err
	}
	isCommited = true
	return nil
}

//refund Failure entry
