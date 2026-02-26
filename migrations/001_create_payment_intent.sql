CREATE TABLE IF NOT EXISTS  payment.payment_intent(
		payment_id UUID PRIMARY KEY,
		idempotency_key TEXT NOT NULL UNIQUE,
		status TEXT NOT NULL CHECK (
			status IN ('CREATED', 'PROCESSING', 'CAPTURED', 'FAILED', 'CANCELLED')
		),
		amount BIGINT NOT NULL CHECK (amount > 0),
		currency TEXT NOT NULL CHECK (char_length(currency) = 3),
		psp_ref_id TEXT NULL UNIQUE,
		psp_name TEXT NOT NULL,
		created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
		updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
		claimed_at TIMESTAMPTZ NULL,

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
        ( -- payment Cancelled
            status = 'CANCELLED'
        )
    )
	);