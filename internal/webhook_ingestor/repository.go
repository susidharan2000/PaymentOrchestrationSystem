package webhookingestor

import (
	"database/sql"
	"fmt"
)

type WebhookRepository interface {
	CreateLedgerEntries() error
	AppendLedger(piID string, paymentID string, paymentStatus string) error
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

func (r *repo) AppendLedger(piID string, paymentID string, paymentStatus string) error {
	return nil
}
