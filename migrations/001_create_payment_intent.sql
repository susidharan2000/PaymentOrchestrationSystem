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
    CONSTRAINT uniq_idempotency UNIQUE (idempotency_key),

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
    CONSTRAINT payment_state_psp_consistency CHECK (
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
CREATE OR REPLACE FUNCTION payment.set_updated_at()
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
EXECUTE FUNCTION payment.set_updated_at();