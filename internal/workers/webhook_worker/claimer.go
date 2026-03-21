package webhookworker

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strconv"
	"time"

	"github.com/stripe/stripe-go/v76"
)

func StartWebhookWorkers(repo workerRepository) {

	workerCount := 2 // default
	if val := os.Getenv("WEBHOOK_WORKER_COUNT"); val != "" {
		if parsed, err := strconv.Atoi(val); err == nil && parsed > 0 {
			workerCount = parsed
		}
	}
	for i := 0; i < workerCount; i++ {
		go worker(repo)
	}
}

//process  the webhook

func worker(repo workerRepository) {
	for {
		processBatch(repo)
		time.Sleep(1 * time.Second)
	}
}

func processBatch(repo workerRepository) {
	events, err := repo.ClaimEvents(10)
	if err != nil {
		log.Println("claim error:", err)
		return
	}
	if len(events) == 0 {
		return
	}
	for _, e := range events {
		//log.Println(e)
		err := processEvent(repo, e)
		if err != nil {
			log.Println("event Handling failed:", err)
			if e.Attempts >= 5 {
				err := repo.MarkEventFailed(e.ID)
				if err != nil {
					log.Println("mark failed error:", err)
				}
			}
		}
	}
}

func processEvent(repo workerRepository, e EventDetails) error {
	tx, err := repo.BeginTx()
	if err != nil {
		return err
	}
	defer func() {
		if r := recover(); r != nil {
			if rbErr := tx.Rollback(); rbErr != nil {
				log.Println("rollback error", rbErr)
			}
			log.Println("panic Recovered", r)
			err = fmt.Errorf("panic %v", r)
		}
	}()

	err = handleEvent(tx, e.Payload, repo)
	if err != nil {
		if rbErr := tx.Rollback(); rbErr != nil {
			log.Println("rollback error", rbErr)
		}
		return err
	}

	err = repo.MarkProcessedTx(tx, e.ID)
	if err != nil {
		if rbErr := tx.Rollback(); rbErr != nil {
			log.Println("rollback error", rbErr)
		}
		return err
	}
	if err := tx.Commit(); err != nil {
		log.Println("commit error:", err)
		return err
	}
	log.Println("webhook event Procesed Successfully")
	log.Println("-------------------------------------------")
	return nil
}

// process stripe event only
func handleEvent(tx *sql.Tx, payload []byte, repo workerRepository) error {
	//verify the Stripe Signature
	var event stripe.Event

	if err := json.Unmarshal(payload, &event); err != nil {
		return err
	}

	log.Println("Processing event:", event.ID)
	switch event.Type {
	case "payment_intent.succeeded", "payment_intent.payment_failed":
		// extract the Payment Intent Details from the event
		var pi stripe.PaymentIntent
		if err := json.Unmarshal(event.Data.Raw, &pi); err != nil {
			log.Println("failed to parse event", err)
			return err
		}
		piID := pi.ID
		currency := pi.Currency
		amount := pi.Amount

		var paymentDetails WebhookPaymentDetails

		paymentDetails.PiID = piID
		paymentDetails.Currency = string(currency)
		paymentDetails.Amount = amount
		paymentDetails.PspName = "stripe"

		log.Printf("PiID: %s", piID)
		switch event.Type {
		case "payment_intent.succeeded":
			//append only in ledger
			if err := repo.RecordPaymentSuccess(paymentDetails, tx); err != nil {
				log.Println("processing failed:", err)
				return err
			}
			log.Println("Payment Webhook Process Completed")
		case "payment_intent.payment_failed":
			//update in payment_intent
			if err := repo.MarkPaymentFailed(paymentDetails, tx); err != nil {
				log.Println("processing failed:", err)
				return err
			}
			log.Println("Payment Webhook Process Completed")
		}
	case "refund.updated":
		var r stripe.Refund
		if err := json.Unmarshal(event.Data.Raw, &r); err != nil {
			log.Println("failed to parse event", err)
			return err
		}
		log.Printf("Refund Status: %s", r.Status)
		switch r.Status {
		case stripe.RefundStatusSucceeded:
			//append only in ledger
			log.Printf("pspRefund_id: %s", r.ID)
			if err := repo.RecordRefundSuccess(r.ID, tx); err != nil {
				log.Println("processing failed:", err)
				return err
			}
			log.Println("Refund Webhook Process Completed")
		case stripe.RefundStatusFailed:
			//update in refund_record
			log.Printf("pspRefund_id: %s", r.ID)
			if err := repo.RecordRefundFailed(r.ID, tx); err != nil {
				log.Println("processing failed:", err)
				return err
			}
			log.Println("Refund Webhook Process Completed")
		default:
			log.Printf("Unhandled refund status: %s", r.Status)
		}
	default:
		log.Printf("Unhandled event type: %s", event.Type)
	}
	return nil
}
