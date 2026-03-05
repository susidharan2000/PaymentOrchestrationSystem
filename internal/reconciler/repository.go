package reconciler

import "database/sql"

type ReconcilerRepository interface {
	ClaimUnresolvedPayments(limit int) ([]Payment, error)
	AppendLedgerEntry(payment Payment, paymentStatus string) error
}

type repo struct {
	db *sql.DB
}

func NewReconcilerRepository(db *sql.DB) ReconcilerRepository {
	return &repo{db: db}
}

func (r *repo) ClaimUnresolvedPayments(limit int) ([]Payment, error) {
	//implement Lease Claim Pattern to claim payment
	// - because it avoids the multiple unnessary psp calls
	// - safe fop multiple reconciler instance
	Payments := make([]Payment, 0, 10)
	query := `WITH rows AS(
		SELECT payment_id 
		FROM payment.payment_intent
		WHERE status IN ('PROCESSING') AND 
		(
			claimed_at IS NULL OR
			claimed_at < now() - interval '30 seconds'
		)
		ORDER BY updated_at ASC, payment_id ASC
		FOR UPDATE SKIP LOCKED
		LIMIT $1
		)
		UPDATE payment.payment_intent p
		SET claimed_at = now()
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
		Payments = append(Payments, payment)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return Payments, nil
}

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
