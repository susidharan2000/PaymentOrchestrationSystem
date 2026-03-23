package eventprojector

import (
	"database/sql"
	"fmt"

	"github.com/lib/pq"
)

type ProjectorRepository interface {
	//projectState() error
	BeginTx() (*sql.Tx, error)
	GetOffsetForUpdate(tx *sql.Tx, projectorName string) (int64, error)
	FetchLedger(tx *sql.Tx, lastSeq int64, limit int64) ([]LedgerEntry, error)
	GetPaymentStates(tx *sql.Tx, paymentIDs []string) (map[string]State, error)
	UpdatePaymentState(tx *sql.Tx, paymentID string, state State, newStatus string) error
	UpdateOffset(tx *sql.Tx, projectorName string, newSeq int64) error
}
type repo struct {
	db *sql.DB
}

func NewProjectorRepository(db *sql.DB) ProjectorRepository {
	return &repo{db: db}
}

func (r *repo) BeginTx() (*sql.Tx, error) {
	return r.db.Begin()
}

func (r *repo) GetOffsetForUpdate(tx *sql.Tx, projectorName string) (int64, error) {
	var lastSeq int64
	err := tx.QueryRow(`SELECT last_processed_seq FROM payment.projector_offsets WHERE projector_name = $1 FOR UPDATE;`, projectorName).Scan(&lastSeq)
	if err != nil {
		if err == sql.ErrNoRows {
			return 0, fmt.Errorf("projector offset not initialized: %s", projectorName)
		}
		return 0, fmt.Errorf("get offset for update failed: %w", err)
	}
	return lastSeq, nil
}

func (r *repo) FetchLedger(tx *sql.Tx, lastSeq int64, limit int64) ([]LedgerEntry, error) {
	entries := make([]LedgerEntry, 0, int(limit))
	rows, err := tx.Query(`SELECT payment_id, seq, entry_type, amount FROM payment.ledger_entries WHERE seq > $1 ORDER BY seq ASC LIMIT $2`, lastSeq, limit)
	if err != nil {
		return nil, fmt.Errorf("fetch ledger query failed: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var e LedgerEntry
		if err := rows.Scan(&e.PaymentID, &e.Seq, &e.EntryType, &e.Amount); err != nil {
			return nil, err
		}
		entries = append(entries, e)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("fetch ledger failed: %w", err)
	}
	return entries, nil
}

func (r *repo) GetPaymentStates(tx *sql.Tx, paymentIDs []string) (map[string]State, error) {
	states := make(map[string]State, len(paymentIDs))

	rows, err := tx.Query(`
        SELECT payment_id, captured_amount, refunded_amount, last_applied_seq
        FROM payment.payment_intent
        WHERE payment_id = ANY($1)
        FOR UPDATE
    `, pq.Array(paymentIDs))
	if err != nil {
		return nil, fmt.Errorf("batch fetch payment states failed: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var paymentID string
		var s State

		if err := rows.Scan(&paymentID, &s.CapturedAmount, &s.RefundedAmount, &s.LastAppliedSeq); err != nil {
			return nil, fmt.Errorf("scan payment state failed: %w", err)
		}

		states[paymentID] = s
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iteration error: %w", err)
	}
	//check all row are present
	for _, id := range paymentIDs {
		if _, ok := states[id]; !ok {
			return nil, fmt.Errorf("payment state not found for id=%s", id)
		}
	}

	return states, nil
}

func (r *repo) UpdatePaymentState(tx *sql.Tx, paymentID string, state State, newStatus string) error {
	res, err := tx.Exec(`
        UPDATE payment.payment_intent
        SET captured_amount = $1,
            refunded_amount = $2,
			last_applied_seq = $3,
            status = $4
        WHERE payment_id = $5
    `, state.CapturedAmount, state.RefundedAmount, state.LastAppliedSeq, newStatus, paymentID)

	if err != nil {
		return fmt.Errorf("update payment state query failed: %w", err)
	}
	rows, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("update payment state rows check failed: %w", err)
	}
	if rows == 0 {
		return fmt.Errorf("update payment state failed for paymentID=%s", paymentID)
	}
	return nil
}

func (r *repo) UpdateOffset(tx *sql.Tx, projectorName string, newSeq int64) error {
	res, err := tx.Exec(`
        UPDATE payment.projector_offsets
        SET last_processed_seq = $1,
            updated_at = now()
        WHERE projector_name = $2
		AND last_processed_seq < $1
    `, newSeq, projectorName)

	if err != nil {
		return fmt.Errorf("update offset query failed: %w", err)
	}

	rows, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("update offset rows check failed: %w", err)
	}

	if rows == 0 {
		return nil
	}
	return nil
}
