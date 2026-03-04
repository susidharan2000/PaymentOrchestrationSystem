package intent

import (
	"database/sql"
	"errors"
	"log"

	"github.com/google/uuid"
	"github.com/susidharan/payment-orchestration-system/internal/domain"
)

type PaymentRepository interface {
	PersistPaymentRequest(req CreatePaymentRequest, requestHash string) (paymentDetails domain.PaymentParams, created bool, err error)
	MarkProcessing(PaymentId string, pspReferenceID string) error
	getPaymentById(PaymentId string) (PaymentDetailsByID, error)
	//CancelPayment(ctx context.Context, paymentID string) error
}
type repo struct {
	db *sql.DB
}

func NewPaymentRepository(db *sql.DB) PaymentRepository {
	return &repo{db: db}
}

// create payment request
func (r *repo) PersistPaymentRequest(req CreatePaymentRequest, requestHash string) (domain.PaymentParams, bool, error) {
	paymentId := generatePaymentID()
	var paymentDetails domain.PaymentParams
	var existingHash string
	var created bool
	query := `INSERT INTO payment.payment_intent (payment_id,idempotency_key,status,amount,currency,psp_name,request_hash) VALUES ($1,$2,'CREATED',$3,$4,$5,$6)
	ON CONFLICT (idempotency_key)
	DO UPDATE SET idempotency_key = EXCLUDED.idempotency_key
	RETURNING payment_id,amount,currency,request_hash,psp_ref_id,(xmax = 0) AS created;`
	err := r.db.QueryRow(query, paymentId, req.IdempotencyKey, req.Amount, req.Currency, req.PspName, requestHash).Scan(&paymentDetails.PaymentId, &paymentDetails.Amount, &paymentDetails.Currency, &existingHash, &paymentDetails.PspRefID, &created)

	if err != nil {
		return domain.PaymentParams{}, created, err
	}
	// if conflict happend
	if existingHash != requestHash {
		return domain.PaymentParams{}, created, errors.New("idempotency key reused with different payload")
	}
	return paymentDetails, created, nil
}

func (r *repo) MarkProcessing(PaymentId string, pspReferenceID string) error {
	row, err := r.db.Exec(`UPDATE payment.payment_intent 
	SET status = 'PROCESSING',psp_ref_id = $1,updated_at = now()
	WHERE payment.payment_intent.payment_id=$2 
	AND payment.payment_intent.status='CREATED'
	AND payment.payment_intent.psp_ref_id IS NULL
	`, pspReferenceID, PaymentId)
	if err != nil {
		return err
	}
	res, _ := row.RowsAffected()
	if res == 0 {
		log.Println("Failed to change the status to PROCESSING", PaymentId)
	}
	return nil
}

// get payment By ID
func (r *repo) getPaymentById(PaymentId string) (PaymentDetailsByID, error) {
	var PaymentDetails PaymentDetailsByID
	err := r.db.QueryRow(`SELECT amount,currency,status,psp_name FROM payment.payment_intent WHERE payment_id=$1`, PaymentId).Scan(&PaymentDetails.Amount, &PaymentDetails.Currency, &PaymentDetails.Status, &PaymentDetails.PspName)
	if err != nil {
		return PaymentDetailsByID{}, err
	}
	return PaymentDetails, err
}

// cancel payment
// func (r *repo) CancelPayment(ctx context.Context, paymentID string) error {
// 	res, err := r.db.ExecContext(ctx, `UPDATE payment.payment_intent
//     SET status = 'CANCELLED',
//     updated_at = now()
//     WHERE payment_id = $1
//     AND status = 'CREATED'
//     AND psp_ref_id IS NULL
//     `, paymentID)
// 	if err != nil {
// 		return err
// 	}
// 	rows, err := res.RowsAffected()
// 	if err != nil {
// 		return err
// 	}
// 	if rows > 0 {
// 		return nil
// 	}
// 	var status string
// 	err = r.db.QueryRowContext(ctx, `SELECT status FROM payment.payment_intent WHERE payment_id = $1`, paymentID).Scan(&status)
// 	if err == sql.ErrNoRows {
// 		return errors.New("payment Not found")
// 	}
// 	if err != nil {
// 		return err
// 	}

// 	switch status {
// 	case "PROCESSING":
// 		return errors.New("payment is processing")
// 	case "CAPTURED":
// 		return errors.New("payment already captured")
// 	case "FAILED":
// 		return errors.New("payment already failed")
// 	case "CANCELLED":
// 		return errors.New("payment already cancelled")
// 	default:
// 		return errors.New("invalid Payment state")
// 	}
// }

func generatePaymentID() string {
	return uuid.NewString()
}
