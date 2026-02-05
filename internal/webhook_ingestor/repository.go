package webhookingestor

import (
	"database/sql"
	"errors"
	"fmt"
	"log"
)

type WebhookRepository interface {
	CreateLedgerEntries() error
	GetPaymentDetails(paymentID string) (PaymemtIntentDetails, error)
	AppendLedger(PaymentDetails PaymemtIntentDetails) error
}
type repo struct {
	db *sql.DB
}

func NewWebhookRepository(db *sql.DB) WebhookRepository {
	return &repo{db: db}
}

// create the table
func (r *repo) CreateLedgerEntries() error {
	if err := r.enableExtensions(); err != nil {
		return fmt.Errorf("enable extensions: %w", err)
	}

	if err := r.createLedgerTable(); err != nil {
		return fmt.Errorf("create ledger table: %w", err)
	}

	if err := r.createLedgerGuardFunction(); err != nil {
		return fmt.Errorf("create ledger guard function: %w", err)
	}

	if err := r.createLedgerTriggers(); err != nil {
		return fmt.Errorf("create ledger triggers: %w", err)
	}
	return nil
}

func (r *repo) GetPaymentDetails(paymentID string) (PaymemtIntentDetails, error) {
	log.Println(paymentID)
	var PaymentDetails PaymemtIntentDetails
	err := r.db.QueryRow(`SELECT amount,currency,psp_name FROM payment.payment_intent WHERE payment_id= $1;`, paymentID).Scan(&PaymentDetails.Amount, &PaymentDetails.Currency, &PaymentDetails.PspName)
	if err != nil {
		return PaymemtIntentDetails{}, err
	}
	return PaymentDetails, nil
}

func (r *repo) AppendLedger(PaymentDetails PaymemtIntentDetails) error {

	row, err := r.db.Exec(`INSERT INTO payment.ledger_entries (entry_type,payment_id,amount,currency,psp_name,psp_ref_id) VALUES ($1,$2,$3,$4,$5,$6)`, PaymentDetails.Status, PaymentDetails.PaymentId, PaymentDetails.Amount, PaymentDetails.Currency, PaymentDetails.PspName, PaymentDetails.PiID)
	if err != nil {
		return err
	}
	res, err := row.RowsAffected()
	if err != nil {
		return err
	}
	if res == 0 {
		return errors.New("ledger append failed")
	}
	return nil
}
