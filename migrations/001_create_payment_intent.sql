CREATE TABLE IF NOT EXISTS  payment.payment_intent(
		payment_id UUID PRIMARY KEY,
		idempotency_key TEXT NOT NULL UNIQUE,
		status TEXT NOT NULL CHECK (
			status IN ('CREATED', 'PROCESSING', 'UNKNOWN', 'CAPTURED', 'FAILED', 'CANCELLED')
		),
		amount BIGINT NOT NULL CHECK (amount > 0),
		currency TEXT NOT NULL,
		psp_ref_id TEXT NULL UNIQUE,
		psp_name TEXT NULL,
		created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
		updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
	);