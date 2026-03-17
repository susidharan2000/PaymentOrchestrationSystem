package worker

import (
	"log"
	"math/rand"
	"os"
	"strconv"
	"time"

	psp "github.com/susidharan/payment-orchestration-system/internal/psp"
)

func StartRefundWorkers(repo workerRepository, registry *psp.Registry) {

	workerCount := 2 // default
	if val := os.Getenv("REFUND_WORKER_COUNT"); val != "" {
		if parsed, err := strconv.Atoi(val); err == nil && parsed > 0 {
			workerCount = parsed
		}
	}

	for i := 0; i < workerCount; i++ {
		go worker(repo, registry)
	}
}

//worker - logic
//Claim a refund payment from refund_record
//Call the external PSP to initiate the refund -- will return psp_refund_id
//Update the refund_record status to "Processing" and update the psp_refund_id
//Worker exits or continues polling for the next refund job

func worker(repo workerRepository, registry *psp.Registry) {
	for {
		//claim the refund job
		refundDetails, err := repo.ClaimRefundablePayment()
		if err != nil {
			//sleep fop 2 seconds
			log.Println("WORKER ERROR")
			time.Sleep(3*time.Second + time.Duration(rand.Intn(2000))*time.Millisecond)
			continue
		}
		if refundDetails.refundID == "" {
			//sleep fop 2 seconds
			//log.Println("no work available")
			time.Sleep(3*time.Second + time.Duration(rand.Intn(2000))*time.Millisecond)
			continue
		}
		log.Printf("processing refund %s", refundDetails.refundID)
		//call the External PSP
		//psp
		provider, ok := registry.Get(refundDetails.PspName)
		if !ok {
			log.Println("unknown PSP:", refundDetails.PspName)
			time.Sleep(3*time.Second + time.Duration(rand.Intn(2000))*time.Millisecond)
			continue
		}
		pspRefundID, err := provider.CreateRefund(refundDetails.pspReferenceID, refundDetails.amount, refundDetails.refundID)
		if err != nil {
			log.Printf("refund %s PSP error: %v", refundDetails.refundID, err)
			continue
		}
		log.Println(pspRefundID)
		//update the payment_intent State
		if err := repo.MarkProcessing(refundDetails.refundID, pspRefundID); err != nil {
			log.Printf("MarkProcessing error: %v", err)
		}
	}
}
