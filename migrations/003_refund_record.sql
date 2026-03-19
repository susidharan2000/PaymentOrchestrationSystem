CREATE EXTENSION IF NOT EXISTS pgcrypto;
CREATE TABLE IF NOT EXISTS payment.refund_record (
	refund_entry_id UUID PRIMARY KEY
				DEFAULT gen_random_uuid(),

    idempotency_key TEXT NOT NULL,

	status TEXT NOT NULL
		CHECK (status IN ('PENDING','PROCESSING','SUCCEEDED','FAILED')),

	payment_id UUID NOT NULL
		REFERENCES payment.payment_intent(payment_id),

	amount BIGINT NOT NULL
		CHECK (amount > 0),

	currency TEXT 
		CHECK (char_length(currency) = 3),

	psp_name TEXT ,
	psp_payment_ref_id TEXT CHECK (length(psp_payment_ref_id) > 0),
    psp_refund_id TEXT CHECK (length(psp_refund_id) > 0),

	-- worker 
	next_retry_at TIMESTAMPTZ NOT NULL DEFAULT now(),
	retry_count INTEGER NOT NULL DEFAULT 0,
	max_retry INTEGER NOT NULL DEFAULT 5,

	-- reconciler
	next_reconcile_at TIMESTAMPTZ NOT NULL DEFAULT now(),
	reconcile_attempts INTEGER DEFAULT 0,
    last_reconcile_error TEXT NULL,

	created_at TIMESTAMPTZ NOT NULL
		DEFAULT now(),
	updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),

    UNIQUE(payment_id,idempotency_key),

	CHECK (retry_count <= max_retry),

	--terminal state must have the psp_refund_id
	CHECK (
        status NOT IN ('SUCCEEDED','FAILED')
        OR psp_refund_id IS NOT NULL
        ),

	-- Refund Consistancy
	CONSTRAINT payment_state_psp_consistency
        CHECK (
		    ( -- Worker domain: Eligible states for worker to claim the refund Job
                status = 'PENDING' 
                AND psp_refund_id IS NULL
            )
		OR 
			(  -- Reconciler domain:  Eligible states for Reconciler to claim the payment
                status IN ('PROCESSING','SUCCEEDED','FAILED')
                AND psp_refund_id IS NOT NULL
            )
	    )
);

-- Trigger for Updated_at

CREATE OR REPLACE FUNCTION payment.set_updated_at()
RETURNS trigger AS $$
BEGIN
    IF row(NEW.*) IS DISTINCT FROM row(OLD.*) THEN
        NEW.updated_at = now();
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER set_updated_at_trigger
BEFORE UPDATE ON payment.refund_record
FOR EACH ROW
EXECUTE FUNCTION payment.set_updated_at();


-- Indexing  
CREATE UNIQUE INDEX uniq_psp_refund_id
ON payment.refund_record(psp_refund_id)
WHERE psp_refund_id IS NOT NULL;

