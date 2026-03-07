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
//concurrently  process the payemnt (max concurrency :5) for v1
// wait untill the all queried payment are resolved

func StartReconciler(repo ReconcilerRepository, r *rand.Rand, registry *psp.Registry) {
	batchSize := 50 // default
	if val := os.Getenv("RECONCILER_BATCH_SIZE"); val != "" {
		if parsed, err := strconv.Atoi(val); err == nil && parsed > 0 {
			batchSize = parsed
		}
	}
	PaymentChan := make(chan Payment, batchSize)
	// concurrently Process Payments and update the ledge
	spanWorkers(repo, PaymentChan, registry)
	for {

		Payments, err := repo.ClaimUnresolvedPayments(batchSize)
		if err != nil {
			log.Printf("Reconciler Error: %s", err)
			return
		}
		log.Println(len(Payments))
		if len(Payments) == 0 {
			time.Sleep(3*time.Second + time.Duration(r.Intn(500))*time.Millisecond)
			continue
		}
		// fill the channel buffer
		for _, payment := range Payments {
			PaymentChan <- payment
		}
	}
}

func spanWorkers(repo ReconcilerRepository, PaymentChan chan Payment, registry *psp.Registry) {
	workerCount := 1 // default
	if val := os.Getenv("RECONCILER_CONCURRENCY"); val != "" {
		if parsed, err := strconv.Atoi(val); err == nil && parsed > 0 {
			workerCount = parsed
		}
	}

	for i := 0; i < workerCount; i++ {
		rn := rand.New(rand.NewSource(time.Now().UnixNano() + int64(i)))
		go processPayment(repo, PaymentChan, rn, registry) //wokers that process payment
	}
}

func processPayment(repo ReconcilerRepository, PaymentChan chan Payment, r *rand.Rand, registry *psp.Registry) {
	for payment := range PaymentChan {
		func() {
			defer func() {
				if rec := recover(); rec != nil {
					log.Printf("panic: %v", rec)
				}
			}()
			//resolve payment
			// query the External PSP ledger and update the internal Ledger
			if payment.PspRefID == nil {
				log.Println("missing psp_ref_id for payment:", payment.PaymentId)
				return
			}

			//psp
			provider, ok := registry.Get(payment.PspName)
			if !ok {
				log.Println("unknown PSP:", payment.PspName)
				return
			}

			var pi domain.PspIntent
			var err error
			var retryable bool
			// implement the Retry for the 429 and 500 error
			for attempt := 0; attempt < 3; attempt++ {
				//pi, err = queryStripeByPSPRef(*payment.PspRefID)
				pi, retryable, err = provider.GetPaymentIntent(*payment.PspRefID)
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
				log.Printf("PSP query failed after retries for payment %s: %v", payment.PaymentId, err)
				return
			}
			log.Printf("PSP %s payment status: %v", payment.PspName, pi.Status)
			switch pi.Status {
			case domain.StatusSucceeded:
				if err := repo.AppendLedgerEntry(payment, "CAPTURED"); err != nil {
					log.Println(err)
				}
			case domain.StatusFailed:
				if err := repo.AppendLedgerEntry(payment, "FAILED"); err != nil {
					log.Println(err)
				}
			default:
			}
		}()
	}
}

// func queryStripeByPSPRef(pspRefID string) (*stripe.PaymentIntent, error) {
// 	//ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
// 	//defer cancel()

// 	params := &stripe.PaymentIntentParams{}
// 	//params.Context = ctx

// 	return paymentintent.Get(pspRefID, params)
// }
