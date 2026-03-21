package webhookingestor

import (
	"database/sql"

	model "github.com/susidharan/payment-orchestration-system/internal/webhook_ingestor/model"
)

type WebhookRepository interface {
	StoreWebhookEvent(eventDetails model.EventDetails) error
}
type repo struct {
	db *sql.DB
}

func NewWebhookRepository(db *sql.DB) WebhookRepository {
	return &repo{db: db}
}

// insert the webhook event
func (r *repo) StoreWebhookEvent(eventDetails model.EventDetails) error {
	_, err := r.db.Exec(`
        INSERT INTO payment.webhook_events (event_id, psp_name, payload)
        VALUES ($1, $2, $3)
        ON CONFLICT (event_id, psp_name) DO NOTHING
    `, eventDetails.EventID, eventDetails.PspName, eventDetails.Payload)
	return err
}
