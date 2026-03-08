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

			currency TEXT NOT NULL
				CHECK (char_length(currency) = 3),

			psp_name TEXT NOT NULL,
			psp_payment_ref_id TEXT NOT NULL CHECK (length(psp_payment_ref_id) > 0),
            psp_refund_id TEXT CHECK (length(psp_refund_id) > 0),

			created_at TIMESTAMPTZ NOT NULL
				DEFAULT now(),

            UNIQUE(idempotency_key),
            UNIQUE(psp_refund_id),

			CHECK (
            status NOT IN ('SUCCEEDED','FAILED')
            OR psp_refund_id IS NOT NULL
            )
    
		);