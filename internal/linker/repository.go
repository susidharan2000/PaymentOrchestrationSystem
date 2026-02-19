package linker

import (
	"database/sql"
)

type LinkerRepository interface {
	linkLedger() (int64, error)
}
type repo struct {
	db *sql.DB
}

func NewLinkerRepository(db *sql.DB) LinkerRepository {
	return &repo{db: db}
}

func (r *repo) linkLedger() (int64, error) {
	tx, err := r.db.Begin()
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()
	// link the ledger
	res, err := tx.Exec(`WITH batch AS(
		SELECT le.ledger_entry_id,pi.payment_id 
		FROM payment.payment_intent pi JOIN payment.ledger_entries le 
		ON pi.psp_ref_id = le.psp_ref_id 
		AND pi.psp_name  = le.psp_name
		WHERE le.payment_id IS NULL
		ORDER BY le.ledger_entry_id
		FOR UPDATE SKIP LOCKED
		LIMIT 100
	)
	UPDATE payment.ledger_entries le
	SET payment_id = batch.payment_id
	FROM batch
	WHERE le.ledger_entry_id = batch.ledger_entry_id;
	`)
	if err != nil {
		//log.Panic(err)
		return 0, err
	}
	rows, err := res.RowsAffected()
	if err != nil {
		return 0, err
	}
	if err := tx.Commit(); err != nil {
		return 0, err
	}
	return rows, err
}
