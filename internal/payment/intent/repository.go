package intent

import (
	"database/sql"

	"github.com/google/uuid"
)

type PaymentRepository interface {
	PersistPaymentRequest(req CreatePaymentRequest) (paymentID string, created bool, err error)
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

func generatePaymentID() string {
	return uuid.NewString()
}
