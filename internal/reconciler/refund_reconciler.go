package reconciler

import (
	"log"
	"math/rand"
	"os"
	"strconv"
	"time"

	domain "github.com/susidharan/payment-orchestration-system/internal/domain"
	psp "github.com/susidharan/payment-orchestration-system/internal/psp"
)

//get the unresolved Payments from the payment_intent (limit 10)
//sleep when len(paymenys) == 0
//concurrently  process the payemnt (max concurrency :5)
// wait untill the all queried payment are resolved

func StartRefundReconciler(repo ReconcilerRepository, r *rand.Rand, registry *psp.Registry) {
	batchSize := 50 // default
	if val := os.Getenv("REFUND_RECONCILER_BATCH_SIZE"); val != "" {
		if parsed, err := strconv.Atoi(val); err == nil && parsed > 0 {
			batchSize = parsed
		}
	}
	refundChan := make(chan Refund, batchSize)
	// concurrently Process Payments and update the ledge
	spanRefundWorkers(repo, refundChan, registry)
	for {

		refundPayments, err := repo.ClaimUnresolvedRefunds(batchSize)
		if err != nil {
			log.Printf("Reconciler Error: %s", err)
			return
		}
		log.Println(len(refundPayments))
		if len(refundPayments) == 0 {
			time.Sleep(3*time.Second + time.Duration(r.Intn(500))*time.Millisecond)
			continue
		}
		// fill the channel buffer
		for _, payment := range refundPayments {
			refundChan <- payment
		}
	}
}

func spanRefundWorkers(repo ReconcilerRepository, refundChan chan Refund, registry *psp.Registry) {
	workerCount := 1 // default
	if val := os.Getenv("REFUND_RECONCILER_CONCURRENCY"); val != "" {
		if parsed, err := strconv.Atoi(val); err == nil && parsed > 0 {
			workerCount = parsed
		}
	}

	for i := 0; i < workerCount; i++ {
		rn := rand.New(rand.NewSource(time.Now().UnixNano() + int64(i)))
		go processRefund(repo, refundChan, rn, registry) //wokers that process payment
	}
}

func processRefund(repo ReconcilerRepository, refundChan chan Refund, r *rand.Rand, registry *psp.Registry) {
	for refundPayment := range refundChan {
		func() {
			defer func() {
				if rec := recover(); rec != nil {
					log.Printf("panic: %v", rec)
				}
			}()
			//resolve payment
			// query the External PSP ledger and update the internal Ledger
			if refundPayment.PspRefundID == nil {
				log.Println("missing psp_refund_id for refund:", refundPayment.RefundID)
				return
			}
			//psp
			provider, ok := registry.Get(refundPayment.PspName)
			if !ok {
				log.Println("unknown PSP:", refundPayment.PspName)
				return
			}

			var refundStatus domain.PaymentStatus
			var err error
			var retryable bool
			// implement the Retry for the 429 and 500 error
			for attempt := 0; attempt < 3; attempt++ {
				//pi, err = queryStripeByPSPRef(*payment.PspRefID)
				refundStatus, retryable, err = provider.GetRefund(*refundPayment.PspRefundID)
				if err == nil {
					break
				}
				if !retryable {
					break
				}
				// 429 - Rate limit error
				// 5xx  - Stripe Internal Error
				// avioding this casue  30s–60s+ delay to resolve payment
				backoff := time.Duration(1<<attempt) * time.Second      // add the exponential time for each retry
				jitter := time.Duration(r.Intn(300)) * time.Millisecond // avoid the sequantial time
				time.Sleep(backoff + jitter)
			}
			if err != nil {
				log.Printf("PSP query failed after retries for Refund %s: %v", refundPayment.RefundID, err)
				return
			}
			log.Printf("PSP %s refund status: %v", refundPayment.PspName, refundStatus)
			switch refundStatus {
			case domain.StatusSucceeded:
				if err := repo.RefundSuccessEntry(refundPayment, "REFUND"); err != nil {
					log.Println(err)
				}
			case domain.StatusFailed:
				if err := repo.RefundSuccessEntry(refundPayment, "REFUND_FAILED"); err != nil {
					log.Println(err)
				}
			default:
				log.Printf("refund still processing: %s", refundPayment.RefundID)
			}
		}()
	}
}
