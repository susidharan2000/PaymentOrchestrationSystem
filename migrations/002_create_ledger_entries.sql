CREATE EXTENSION IF NOT EXISTS pgcrypto;
CREATE TABLE IF NOT EXISTS payment.ledger_entries (
	        seq BIGSERIAL PRIMARY KEY,
			ledger_entry_id UUID NOT NULL
				DEFAULT gen_random_uuid()
				UNIQUE,

			entry_type TEXT NOT NULL
				CHECK (entry_type IN ('PAYMENT','REFUND')),

			payment_id UUID NOT NULL
				REFERENCES payment.payment_intent(payment_id),

			refund_id UUID
                REFERENCES payment.refund_record(refund_entry_id),

			amount BIGINT NOT NULL
				CHECK (amount > 0),

			currency TEXT NOT NULL
				CHECK (char_length(currency) = 3),

			psp_name TEXT NOT NULL,
			psp_ref_id TEXT NOT NULL CHECK (length(psp_ref_id) > 0),

			created_at TIMESTAMPTZ NOT NULL
				DEFAULT now(),

			UNIQUE (psp_name, psp_ref_id),

			CHECK (
                (entry_type IN ('REFUND') AND refund_id IS NOT NULL)
            OR
            (entry_type IN ('PAYMENT') AND refund_id IS NULL)
            )
		);



CREATE OR REPLACE FUNCTION payment.guard_ledger_mutation()
RETURNS trigger AS $$
BEGIN
    RAISE EXCEPTION 'ledger_entries is append-only';
END;
$$ LANGUAGE plpgsql;


CREATE TRIGGER no_ledger_update
BEFORE UPDATE ON payment.ledger_entries
FOR EACH ROW
EXECUTE FUNCTION payment.guard_ledger_mutation();

CREATE TRIGGER no_ledger_delete
BEFORE DELETE ON payment.ledger_entries
FOR EACH ROW
EXECUTE FUNCTION payment.guard_ledger_mutation();




CREATE INDEX idx_ledger_payment
ON payment.ledger_entries(payment_id);
