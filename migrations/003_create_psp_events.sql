CREATE TABLE IF NOT EXISTS  payment.psp_events(
		id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
        psp_name TEXT NOT NULL,
		psp_event_id TEXT NOT NULL,
        psp_event_type TEXT NOT NULL,
        raw_payload JSONB NOT NULL,
        received_at TIMESTAMPTZ NOT NULL DEFAULT now(),
        UNIQUE (psp_name, psp_event_id)
	);