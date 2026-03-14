CREATE TABLE IF NOT EXISTS  payment.payment_intent(
		payment_id UUID PRIMARY KEY,
		idempotency_key TEXT NOT NULL,
		status TEXT NOT NULL CHECK (
			status IN ('CREATED', 'PROCESSING', 'CAPTURED', 'FAILED', 'CANCELLED', 'PARTIALLY_REFUNDED','REFUNDED')
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

        CONSTRAINT uniq_idempotency UNIQUE (idempotency_key),

        -- Refund State Consistancy
        CONSTRAINT refund_state_consistency CHECK (
        status != 'PARTIALLY_REFUNDED'
        OR refunded_amount > 0
        ),

        CONSTRAINT refund_full_consistency CHECK (
        status != 'REFUNDED'
        OR refunded_amount = captured_amount
        ),
        
        -- Payment Consistancy
		CONSTRAINT payment_state_psp_consistency
        CHECK (
        ( -- Worker domain: Eligible states for worker to claim the payment
            status = 'CREATED' 
            AND psp_ref_id IS NULL
        )
        OR
        (  -- Reconciler domain:  Eligible states for Reconciler to claim the payment
            status = 'PROCESSING'
            AND psp_ref_id IS NOT NULL
        )
        OR
        ( -- Terminal State
            status IN ('CAPTURED','FAILED')
            AND psp_ref_id IS NOT NULL
        )
        OR
        (  -- Post-capture refund states
        status IN ('PARTIALLY_REFUNDED','REFUNDED')
        AND psp_ref_id IS NOT NULL
        )
        OR
        ( -- payment Cancelled
            status = 'CANCELLED'
        )
    )
	);


-- Index
CREATE INDEX idx_reconcile_ready
ON payment.payment_intent(next_reconcile_at,payment_id)
WHERE status = 'PROCESSING';

CREATE UNIQUE INDEX uniq_psp_reference
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

CREATE TRIGGER payment_intent_updated_at
BEFORE UPDATE ON payment.payment_intent
FOR EACH ROW
EXECUTE FUNCTION payment.set_updated_at();