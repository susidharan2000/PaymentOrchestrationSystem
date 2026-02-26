package reconciler

import (
	"context"
	"log"
	"math/rand"
	"os"
	"strconv"
	"sync"
	"time"

	"github.com/stripe/stripe-go/v78"
	"github.com/stripe/stripe-go/v78/paymentintent"
)

//get the unresolved Payments from the payment_intent (limit 10)
//sleep when len(paymenys) == 0
//concurrently  process the payemnt (max concurrency :5) for v1
// wait untill the all queried payment are resolved

func StartReconciler(repo ReconcilerRepository, r *rand.Rand) {
	batchSize := 1 // default
	if val := os.Getenv("RECONCILER_BATCH_SIZE"); val != "" {
		if parsed, err := strconv.Atoi(val); err == nil && parsed > 0 {
			batchSize = parsed
		}
	}
	wg := sync.WaitGroup{}
	PaymentChan := make(chan Payment, batchSize)
	// concurrently Process Payments and update the ledge
	spanWorkers(repo, &wg, PaymentChan)
	for {

		Payments, err := repo.ClaimUnresolvedPayments(10)
		if err != nil {
			log.Printf("Reconciler Error: %s", err)
			return
		}
		log.Println(len(Payments))
		if len(Payments) == 0 {
			time.Sleep(5*time.Second + time.Duration(r.Intn(500))*time.Millisecond)
			continue
		}
		// fill the channel buffer
		for _, payment := range Payments {
			wg.Add(1)
			PaymentChan <- payment
		}
		wg.Wait()
	}
}

func spanWorkers(repo ReconcilerRepository, wg *sync.WaitGroup, PaymentChan chan Payment) {
	workerCount := 1 // default
	if val := os.Getenv("RECONCILER_CONCURRENCY"); val != "" {
		if parsed, err := strconv.Atoi(val); err == nil && parsed > 0 {
			workerCount = parsed
		}
	}

	for i := 0; i < workerCount; i++ {
		rn := rand.New(rand.NewSource(time.Now().UnixNano()))
		go processPayment(repo, wg, PaymentChan, rn) //wokers that process payment
	}
}

func processPayment(repo ReconcilerRepository, wg *sync.WaitGroup, PaymentChan chan Payment, r *rand.Rand) {
	for payment := range PaymentChan {
		func() {
			defer func() {
				if rec := recover(); rec != nil {
					log.Printf("panic: %v", rec)
				}
				wg.Done()
			}()
			//resolve payment
			// query the External PSP ledger and update the internal Ledger
			if payment.PspRefID == nil {
				log.Println("missing psp_ref_id for payment:", payment.PaymentId)
				return
			}
			var pi *stripe.PaymentIntent
			var err error
			// implement the Retry for the 429 and 500 error
			for attempt := 0; attempt < 3; attempt++ {
				pi, err = queryStripeByPSPRef(*payment.PspRefID)
				if err == nil {
					break
				}
				// 429 - Rate limit error
				// 5xx  - Stripe Internal Error
				// avioding this casue  30s–60s+ delay to resolve payment
				if stripeErr, ok := err.(*stripe.Error); ok {
					if stripeErr.HTTPStatusCode == 429 || stripeErr.HTTPStatusCode >= 500 {
						backoff := time.Duration(1<<attempt) * time.Second      // add the exponential time for each retry
						jitter := time.Duration(r.Intn(300)) * time.Millisecond // avoid the sequantial time
						time.Sleep(backoff + jitter)
						continue
					}
				}
				return
			}
			if err != nil {
				log.Printf("stripe call failed after retries: %v", err)
				return
			}
			if pi == nil {
				log.Println("stripe returned nil payment intent")
				return
			}

			switch pi.Status {
			case stripe.PaymentIntentStatusSucceeded:
			case stripe.PaymentIntentStatusCanceled:
			case stripe.PaymentIntentStatusRequiresPaymentMethod:
			default:
			}
		}()
	}
}

func queryStripeByPSPRef(pspRefID string) (*stripe.PaymentIntent, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	params := &stripe.PaymentIntentParams{}
	params.Context = ctx

	return paymentintent.Get(pspRefID, params)
}
