package worker

import (
	"database/sql"
	"log"

	"github.com/susidharan/payment-orchestration-system/internal/domain"
)

type workerRepository interface {
	ClaimPayment() (paymentDetails domain.PaymentParams, err error)
	MarkProcessing(PaymentId string, pspReferenceID string) error
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
		AND psp_ref_id IS NULL 
		AND (claimed_at IS NULL OR claimed_at < now() - interval '30 seconds')
		FOR UPDATE SKIP LOCKED
		LIMIT 1
	)
	UPDATE payment.payment_intent
	SET claimed_at = now()
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

func (r *repo) MarkProcessing(PaymentId string, pspReferenceID string) error {
	row, err := r.db.Exec(`UPDATE payment.payment_intent 
	SET status = 'PROCESSING',claimed_at = NULL ,psp_ref_id = $1,updated_at = now()
	WHERE payment.payment_intent.payment_id=$2 
	AND payment.payment_intent.status='CREATED'
	AND payment.payment_intent.psp_ref_id IS NULL
	AND payment.payment_intent.claimed_at IS NOT NULL
	`, pspReferenceID, PaymentId)
	if err != nil {
		return err
	}
	res, _ := row.RowsAffected()
	if res == 0 {
		log.Println("state transition skipped — likely already updated", PaymentId)
	}
	return nil
}
