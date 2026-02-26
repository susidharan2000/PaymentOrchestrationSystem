package intent

import (
	"context"
	"database/sql"
	"errors"

	"github.com/google/uuid"
)

type PaymentRepository interface {
	PersistPaymentRequest(req CreatePaymentRequest) (paymentID string, created bool, err error)
	CancelPayment(ctx context.Context, paymentID string) error
}
type repo struct {
	db *sql.DB
}

func NewPaymentRepository(db *sql.DB) PaymentRepository {
	return &repo{db: db}
}

// create payment request
func (r *repo) PersistPaymentRequest(req CreatePaymentRequest) (string, bool, error) {
	created := true
	paymentId := generatePaymentID()
	query := `INSERT INTO payment.payment_intent (payment_id,idempotency_key,status,amount,currency,psp_name) VALUES ($1,$2,'CREATED',$3,$4,$5)
	ON CONFLICT (idempotency_key)
	DO NOTHING
	RETURNING payment_id;`
	var id string
	err := r.db.QueryRow(query, paymentId, req.IdempotencyKey, req.Amount, req.Currency, req.PspName).Scan(&id)
	if err == sql.ErrNoRows {
		err = r.db.QueryRow(`SELECT payment_id FROM payment.payment_intent WHERE idempotency_key=$1`, req.IdempotencyKey).Scan(&id)
		created = false
		if err != nil {
			return "", created, err
		}
	}
	if err != nil {
		return "", created, err
	}
	return id, created, nil
}

// cancel payment
func (r *repo) CancelPayment(ctx context.Context, paymentID string) error {
	res, err := r.db.ExecContext(ctx, `UPDATE payment.payment_intent
    SET status = 'CANCELLED',
    updated_at = now()
    WHERE payment_id = $1
    AND status = 'CREATED'
    AND psp_ref_id IS NULL
    `, paymentID)
	if err != nil {
		return err
	}
	rows, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if rows > 0 {
		return nil
	}
	var status string
	err = r.db.QueryRowContext(ctx, `SELECT status FROM payment.payment_intent WHERE payment_id = $1`, paymentID).Scan(&status)
	if err == sql.ErrNoRows {
		return errors.New("payment Not found")
	}
	if err != nil {
		return err
	}

	switch status {
	case "PROCESSING":
		return errors.New("payment is processing")
	case "CAPTURED":
		return errors.New("payment already captured")
	case "FAILED":
		return errors.New("payment already failed")
	case "CANCELLED":
		return errors.New("payment already cancelled")
	default:
		return errors.New("invalid Payment state")
	}
}

func generatePaymentID() string {
	return uuid.NewString()
}
