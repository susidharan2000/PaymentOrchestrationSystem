CREATE SCHEMA IF NOT EXISTS payment;
CREATE EXTENSION IF NOT EXISTS pgcrypto;
-- Payment_intent Table
CREATE TABLE IF NOT EXISTS payment.payment_intent(
    payment_id UUID PRIMARY KEY,
    idempotency_key TEXT NOT NULL,
    
    status TEXT NOT NULL CHECK (
        status IN (
            'CREATED','PROCESSING','CAPTURED',
            'FAILED','CANCELLED',
            'PARTIALLY_REFUNDED','REFUNDED'
        )
    ),

    amount BIGINT NOT NULL CHECK (amount > 0),
    captured_amount BIGINT NOT NULL DEFAULT 0,
    refunded_amount BIGINT NOT NULL DEFAULT 0,

    currency TEXT NOT NULL CHECK (char_length(currency) = 3),

    psp_ref_id TEXT NULL,
    psp_name TEXT NOT NULL,

    request_hash TEXT NOT NULL,

    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),

    reconcile_attempts INT DEFAULT 0,
    next_reconcile_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    last_reconcile_error TEXT NULL,

    last_applied_seq BIGINT NOT NULL DEFAULT 0,

    -- Idempotency
    CONSTRAINT uniq_idempotency UNIQUE (psp_name, idempotency_key),

    -- Financial safety 
    CONSTRAINT captured_amount_limit CHECK (
        captured_amount <= amount
    ),
    CONSTRAINT refund_limit CHECK (
        refunded_amount <= captured_amount
    ),
    CONSTRAINT non_negative_amounts CHECK (
        captured_amount >= 0 AND refunded_amount >= 0
    ),

    -- Refund consistency
    CONSTRAINT refund_state_consistency CHECK (
        status != 'PARTIALLY_REFUNDED'
        OR refunded_amount > 0
    ),
    CONSTRAINT refund_full_consistency CHECK (
        status != 'REFUNDED'
        OR refunded_amount = captured_amount
    ),

    -- PSP and state consistency
    CONSTRAINT payment_intent_psp_consistency CHECK (
        (status = 'CREATED' AND psp_ref_id IS NULL)
        OR
        (status = 'PROCESSING' AND psp_ref_id IS NOT NULL)
        OR
        (status IN ('CAPTURED','FAILED') AND psp_ref_id IS NOT NULL)
        OR
        (status IN ('PARTIALLY_REFUNDED','REFUNDED') AND psp_ref_id IS NOT NULL)
        OR
        (status = 'CANCELLED')
    ),

    -- money consistency
    CONSTRAINT status_amount_consistency CHECK (
        (
            status = 'REFUNDED'
            AND refunded_amount = captured_amount
        )
        OR
        (
            status = 'PARTIALLY_REFUNDED'
            AND refunded_amount > 0
            AND refunded_amount < captured_amount
        )
        OR
        (
            status = 'CAPTURED'
            AND captured_amount > 0
            AND refunded_amount = 0
        )
        OR
        (
            status IN ('CREATED','PROCESSING','FAILED','CANCELLED')
        )
    ),
    -- Seq sanity
    CONSTRAINT seq_non_negative CHECK (
        last_applied_seq >= 0
    )
);

-- Index
CREATE INDEX IF NOT EXISTS idx_reconcile_ready
ON payment.payment_intent(next_reconcile_at, payment_id)
WHERE status = 'PROCESSING';

CREATE UNIQUE INDEX IF NOT EXISTS uniq_psp_reference
ON payment.payment_intent(psp_name, psp_ref_id)
WHERE psp_ref_id IS NOT NULL;


-- Trigger For update
CREATE OR REPLACE FUNCTION payment.set_updated_at_payment_intent()
RETURNS trigger AS $$
BEGIN
    NEW.updated_at = now();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS payment_intent_updated_at ON payment.payment_intent;

CREATE TRIGGER payment_intent_updated_at
BEFORE UPDATE ON payment.payment_intent
FOR EACH ROW
EXECUTE FUNCTION payment.set_updated_at_payment_intent();


-- Refund_Record

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
	CONSTRAINT refund_record_psp_consistency
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

CREATE OR REPLACE FUNCTION payment.set_updated_at_generic()
RETURNS trigger AS $$
BEGIN
    IF row(NEW.*) IS DISTINCT FROM row(OLD.*) THEN
        NEW.updated_at = now();
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS refund_record_updated_at ON payment.refund_record;

CREATE TRIGGER refund_record_updated_at
BEFORE UPDATE ON payment.refund_record
FOR EACH ROW
EXECUTE FUNCTION payment.set_updated_at_generic();


-- Indexing  
CREATE UNIQUE INDEX uniq_psp_refund_id
ON payment.refund_record(psp_refund_id)
WHERE psp_refund_id IS NOT NULL;

CREATE INDEX idx_refund_payment
ON payment.refund_record(payment_id);

-- Ledger_Entries

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

			UNIQUE (psp_name, psp_ref_id, entry_type),

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


--Webhook_event
CREATE TABLE IF NOT EXISTS payment.webhook_events (
    id BIGSERIAL PRIMARY KEY,
    event_id TEXT NOT NULL,
    psp_name TEXT NOT NULL,

    payload JSONB NOT NULL,

    status TEXT NOT NULL DEFAULT 'PENDING'
    CHECK (status IN ('PENDING','PROCESSING','PROCESSED','FAILED')),

    attempts INT NOT NULL DEFAULT 0,
    next_retry_at TIMESTAMPTZ NOT NULL DEFAULT now(),

    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),

    UNIQUE (event_id, psp_name)
);

CREATE INDEX idx_webhook_event_pending 
ON payment.webhook_events (status, next_retry_at);


-- trigger
CREATE OR REPLACE FUNCTION payment.update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = now();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS webhook_events_updated_at ON payment.webhook_events;

CREATE TRIGGER webhook_events_updated_at
BEFORE UPDATE ON payment.webhook_events
FOR EACH ROW
EXECUTE FUNCTION payment.update_updated_at_column();

--  Indexing
CREATE INDEX idx_webhook_event_id
ON payment.webhook_events(event_id);

-- projector_offsets

CREATE TABLE IF NOT EXISTS payment.projector_offsets (
    projector_name TEXT PRIMARY KEY,
    last_processed_seq BIGINT NOT NULL DEFAULT 0,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

INSERT INTO payment.projector_offsets (projector_name) VALUES ('payment_projector') ON CONFLICT DO NOTHING;
