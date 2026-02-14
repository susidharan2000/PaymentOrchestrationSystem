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
	workerCount := 5 //Default
	workerCount, err := strconv.Atoi(os.Getenv("WORKER_COUNT"))
	if err != nil {
		log.Print(err)
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
			time.Sleep(time.Second * 10)
			continue
		}
		if err != nil {
			log.Println(err)
			continue
		}
		//log.Println(paymentDetails)
		//call the External PSP
		pspRefID, err := stripeclient.CreatePaymentIntent(paymentDetails)
		//log.Println(pspRefID)
		//update the payment_intent State
		if err != nil {
			if err := repo.MarkUnknown(paymentDetails.PaymentId, ""); err != nil {
				log.Println(err)
			}

		} else {
			if err := repo.MarkUnknown(paymentDetails.PaymentId, pspRefID); err != nil {
				log.Println(err)
			}
		}

	}
}
