CREATE TABLE IF NOT EXISTS payment.projector_offsets (
    projector_name TEXT PRIMARY KEY,
    last_processed_seq BIGINT NOT NULL DEFAULT 0,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

INSERT INTO payment.projector_offsets (projector_name) VALUES ('payment_projector') ON CONFLICT DO NOTHING;