package worker

import (
	"database/sql"

	"github.com/susidharan/payment-orchestration-system/internal/domain"
)

type workerRepository interface {
	ClaimPayment() (paymentDetails domain.PaymentParams, err error)
}

type repo struct {
	db *sql.DB
}

// constructure func
func NewWorkerRepository(db *sql.DB) workerRepository {
	return &repo{db: db}
}
func (r *repo) ClaimPayment() (domain.PaymentParams, error) {
	var paymentDetails domain.PaymentParams
	tx, err := r.db.Begin()
	if err != nil {
		return domain.PaymentParams{}, err
	}
	committed := false
	defer func() {
		if !committed {
			tx.Rollback()
		}
	}()
	err = tx.QueryRow(`WITH row AS (
		SELECT payment_id
		FROM payment.payment_intent
		WHERE status = 'CREATED'
		ORDER BY created_at
		FOR UPDATE SKIP LOCKED
		LIMIT 1
	)
	UPDATE payment.payment_intent
	SET status = 'PROCESSING'
	FROM row
	WHERE payment.payment_intent.payment_id = row.payment_id
	RETURNING payment.payment_intent.payment_id,payment.payment_intent.amount,payment.payment_intent.currency;
	`).Scan(&paymentDetails.PaymentId, &paymentDetails.Amount, &paymentDetails.Currency)
	if err != nil {
		return domain.PaymentParams{}, err
	}
	if err := tx.Commit(); err != nil {
		return domain.PaymentParams{}, err
	}
	committed = true
	return paymentDetails, nil
}
