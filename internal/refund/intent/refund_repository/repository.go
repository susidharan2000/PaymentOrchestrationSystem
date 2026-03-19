package refundrepository

import (
	"database/sql"
	"errors"

	refund_model "github.com/susidharan/payment-orchestration-system/internal/refund/intent/model"
)

type RefundRepository interface {
	CreateRefundRecord(req refund_model.RefundRequest, paymentID string) (refund_model.RefundDetails, bool, error)
}

type repo struct {
	db *sql.DB
}

func NewRefundRepository(db *sql.DB) RefundRepository {
	return &repo{db: db}
}

// Atomic
// case 1: New Refund Request
//   - create refund intent in the refund_record Table
//   - if recoed not Exist then validate the refund request in paymet_intentTable
//   - if valid then update the payment_intnet then update the refund_record and finally return
//   - if not valid then the return payment_not_refundable
//
// case 2: Retry Refund Request
//   - try create refund intent in the refund_record Table
//   - if the record already exist then return the refund Details
func (r *repo) CreateRefundRecord(req refund_model.RefundRequest, paymentID string) (refund_model.RefundDetails, bool, error) {
	var refundDetails refund_model.RefundDetails
	tx, err := r.db.Begin()
	if err != nil {
		return refundDetails, false, err
	}
	defer tx.Rollback()
	//Lock the payment
	var paymentDetails refund_model.PaymentDetails
	row := tx.QueryRow(`SELECT payment_id, currency, psp_name, psp_ref_id FROM payment.payment_intent WHERE payment_id=$1;`, paymentID)
	if err := row.Scan(&paymentDetails.PaymentId, &paymentDetails.Currency, &paymentDetails.PspName, &paymentDetails.PspRefID); err != nil {
		return refund_model.RefundDetails{}, false, err
	}

	// Insert to the Refund Record
	row = tx.QueryRow(`INSERT INTO payment.refund_record (idempotency_key, payment_id, amount, status) VALUES ($1,$2,$3,'PENDING') ON CONFLICT (payment_id, idempotency_key) DO NOTHING RETURNING refund_entry_id;`, req.Idempotencykey, paymentID, req.Amount)
	err = row.Scan(&refundDetails.RefundID)
	if err == sql.ErrNoRows {
		//query with the idempotency_key and get the refund Details
		row := tx.QueryRow(`SELECT refund_entry_id,payment_id,amount,currency,status,created_at FROM payment.refund_record WHERE idempotency_key=$1 LIMIT 1;`, req.Idempotencykey)
		if err := row.Scan(&refundDetails.RefundID, &refundDetails.PaymentID, &refundDetails.Amount, &refundDetails.Currency, &refundDetails.Status, &refundDetails.CreatedAt); err != nil {
			return refund_model.RefundDetails{}, false, err
		}
		if refundDetails.Amount != req.Amount || refundDetails.PaymentID != paymentID {
			return refund_model.RefundDetails{}, false, errors.New("idempotency_conflict")
		}
		return refundDetails, false, nil
	}
	if err != nil {
		return refund_model.RefundDetails{}, false, err
	}

	//validate the refund Request
	var (
		capturedAmount      int64
		refundedAmount      int64
		pendingRefunds      int64
		remainingRefundable int64
	)
	row = tx.QueryRow(`SELECT
    COALESCE(SUM(amount) FILTER (WHERE entry_type = 'PAYMENT'), 0),
    COALESCE(SUM(amount) FILTER (WHERE entry_type = 'REFUND'), 0),
    (
        SELECT COALESCE(SUM(amount), 0)
        FROM payment.refund_record
        WHERE payment_id = $1
        AND status IN ('PENDING','PROCESSING')
		AND refund_entry_id != $2
    )
    FROM payment.ledger_entries
    WHERE payment_id = $1;`, paymentID, refundDetails.RefundID)
	if err = row.Scan(&capturedAmount, &refundedAmount, &pendingRefunds); err != nil {
		return refund_model.RefundDetails{}, false, err
	}
	remainingRefundable = capturedAmount - refundedAmount - pendingRefunds
	if req.Amount > remainingRefundable {
		return refund_model.RefundDetails{}, false, errors.New("payment_not_refundable")
	}

	// Update to the Refund Record
	row = tx.QueryRow(`UPDATE payment.refund_record SET psp_payment_ref_id=$1, psp_name=$2,currency=$3 WHERE refund_entry_id=$4 RETURNING payment_id, amount, currency, status, created_at`,
		paymentDetails.PspRefID,
		paymentDetails.PspName,
		paymentDetails.Currency,
		refundDetails.RefundID,
	)
	if err = row.Scan(
		&refundDetails.PaymentID,
		&refundDetails.Amount,
		&refundDetails.Currency,
		&refundDetails.Status,
		&refundDetails.CreatedAt,
	); err != nil {
		return refund_model.RefundDetails{}, false, err
	}
	err = tx.Commit()
	if err != nil {
		return refund_model.RefundDetails{}, false, err
	}
	return refundDetails, true, nil
}
