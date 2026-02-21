package stateprojector

import (
	"database/sql"
)

type ProjectorRepository interface {
	projectState() error
}
type repo struct {
	db *sql.DB
}

func NewProjectorRepository(db *sql.DB) ProjectorRepository {
	return &repo{db: db}
}

func (r *repo) projectState() error {
	_, err := r.db.Exec(`UPDATE payment.payment_intent p
	SET status = CASE 
	WHEN EXISTS (SELECT 1 FROM payment.ledger_entries le WHERE p.payment_id = le.payment_id AND le.entry_type = 'CAPTURED') THEN 'CAPTURED'
	WHEN EXISTS (SELECT 1 FROM payment.ledger_entries le WHERE p.payment_id = le.payment_id AND le.entry_type = 'FAILED') THEN 'FAILED'
	ELSE p.status
	END
	WHERE p.status IN ('PROCESSING','UNKNOWN')
	`)
	if err != nil {

		return err
	}
	return nil
}
