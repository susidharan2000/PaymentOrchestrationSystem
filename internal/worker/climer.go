package worker

import (
	"database/sql"
	"log"
	"os"
	"strconv"
	"time"

	stripeclient "github.com/susidharan/payment-orchestration-system/internal/psp/stripe"
)

func StartWorkers(repo workerRepository) {

	workerCount := 5 // default
	if val := os.Getenv("WORKER_COUNT"); val != "" {
		if parsed, err := strconv.Atoi(val); err == nil && parsed > 0 {
			workerCount = parsed
		}
	}

	for i := 0; i < workerCount; i++ {
		go worker(repo)
	}
}

func worker(repo workerRepository) {
	for {
		//claim the payment
		paymentDetails, err := repo.ClaimPayment()
		if err == sql.ErrNoRows {
			//sleep fop 2 seconds
			//log.Println("no work available")
			time.Sleep(time.Second * 5)
			continue
		}
		if err != nil {
			log.Println("claim error:", err)
			time.Sleep(2 * time.Second)
			continue
		}
		//log.Println(paymentDetails)
		//call the External PSP
		pspRefID, err := stripeclient.CreatePaymentIntent(paymentDetails)
		if err != nil {
			log.Printf("Error in Worker calling External PSP: %s", err)
			continue
		}
		//log.Println(pspRefID)
		//update the payment_intent State
		if err := repo.MarkProcessing(paymentDetails.PaymentId, pspRefID); err != nil {
			log.Printf("MarkProcessing error: %v", err)
		}
	}
}
