package webhookingestor

func (r *repo) enableExtensions() error {
	_, err := r.db.Exec(`
		CREATE EXTENSION IF NOT EXISTS pgcrypto;
	`)
	return err
}
func (r *repo) createLedgerTable() error {
	_, err := r.db.Exec(`
		CREATE TABLE IF NOT EXISTS payment.ledger_entries (
			ledger_entry_id UUID PRIMARY KEY
				DEFAULT gen_random_uuid(),

			entry_type TEXT NOT NULL
				CHECK (entry_type IN ('CAPTURED', 'REFUND')),

			payment_id UUID NOT NULL
				REFERENCES payment.payment_intent(payment_id),

			amount BIGINT NOT NULL
				CHECK (amount > 0),

			currency TEXT NOT NULL
				CHECK (char_length(currency) = 3),

			psp_name TEXT NOT NULL,
			psp_ref_id TEXT NOT NULL,

			created_at TIMESTAMPTZ NOT NULL
				DEFAULT now(),

			UNIQUE (psp_name, psp_ref_id, entry_type)
		);
	`)
	return err
}
func (r *repo) createLedgerGuardFunction() error {
	_, err := r.db.Exec(`
		CREATE OR REPLACE FUNCTION payment.forbid_ledger_mutation()
		RETURNS trigger AS $$
		BEGIN
			RAISE EXCEPTION 'ledger_entries is append-only';
		END;
		$$ LANGUAGE plpgsql;
	`)
	return err
}
func (r *repo) createLedgerTriggers() error {
	_, err := r.db.Exec(`
		DO $$
		BEGIN
			IF NOT EXISTS (
				SELECT 1 FROM pg_trigger WHERE tgname = 'no_ledger_update'
			) THEN
				CREATE TRIGGER no_ledger_update
				BEFORE UPDATE ON payment.ledger_entries
				FOR EACH ROW
				EXECUTE FUNCTION payment.forbid_ledger_mutation();
			END IF;

			IF NOT EXISTS (
				SELECT 1 FROM pg_trigger WHERE tgname = 'no_ledger_delete'
			) THEN
				CREATE TRIGGER no_ledger_delete
				BEFORE DELETE ON payment.ledger_entries
				FOR EACH ROW
				EXECUTE FUNCTION payment.forbid_ledger_mutation();
			END IF;
		END $$;
	`)
	return err
}
