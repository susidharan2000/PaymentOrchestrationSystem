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

CREATE TRIGGER set_updated_at
BEFORE UPDATE ON payment.webhook_events
FOR EACH ROW
EXECUTE FUNCTION payment.update_updated_at_column();



