CREATE EXTENSION IF NOT EXISTS pgcrypto;
CREATE TABLE IF NOT EXISTS payment.ledger_entries (
			ledger_entry_id UUID PRIMARY KEY
				DEFAULT gen_random_uuid(),

			entry_type TEXT NOT NULL
				CHECK (entry_type IN ('CAPTURED','FAILED')),

			payment_id UUID NULL
				REFERENCES payment.payment_intent(payment_id),

			amount BIGINT NOT NULL
				CHECK (amount > 0),

			currency TEXT NOT NULL
				CHECK (char_length(currency) = 3),

			psp_name TEXT NOT NULL,
			psp_ref_id TEXT NOT NULL CHECK (length(psp_ref_id) > 0),

			created_at TIMESTAMPTZ NOT NULL
				DEFAULT now(),

			UNIQUE (psp_name, psp_ref_id)
		);

CREATE OR REPLACE FUNCTION payment.guard_ledger_mutation()
RETURNS trigger AS $$
BEGIN
    IF TG_OP = 'UPDATE' THEN
        IF OLD.payment_id IS NULL
           AND NEW.payment_id IS NOT NULL
           AND OLD.entry_type = NEW.entry_type
           AND OLD.amount = NEW.amount
           AND OLD.currency = NEW.currency
           AND OLD.psp_name = NEW.psp_name
           AND OLD.psp_ref_id = NEW.psp_ref_id
        THEN
            RETURN NEW;
        END IF;

        RAISE EXCEPTION 'ledger_entries is append-only';
    END IF;

    RAISE EXCEPTION 'ledger_entries is append-only';
END;
$$ LANGUAGE plpgsql;


DROP TRIGGER IF EXISTS no_ledger_update ON payment.ledger_entries;
DROP TRIGGER IF EXISTS no_ledger_delete ON payment.ledger_entries;

CREATE TRIGGER no_ledger_update
BEFORE UPDATE ON payment.ledger_entries
FOR EACH ROW
EXECUTE FUNCTION payment.guard_ledger_mutation();

CREATE TRIGGER no_ledger_delete
BEFORE DELETE ON payment.ledger_entries
FOR EACH ROW
EXECUTE FUNCTION payment.guard_ledger_mutation();
