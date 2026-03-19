-- only to handle idempotent webhook handling
CREATE TABLE IF NOT EXISTS payment.webhook_events (
    id BIGSERIAL PRIMARY KEY,
    event_id TEXT NOT NULL,
    psp_name TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),

    UNIQUE (event_id, psp_name)
);