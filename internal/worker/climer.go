package worker

// import (
// 	"database/sql"
// 	"log"
// 	"os"
// 	"strconv"
// 	"time"
// )

// func StartRefundWorkers(repo workerRepository) {

// 	workerCount := 2 // default
// 	if val := os.Getenv("WORKER_COUNT"); val != "" {
// 		if parsed, err := strconv.Atoi(val); err == nil && parsed > 0 {
// 			workerCount = parsed
// 		}
// 	}

// 	for i := 0; i < workerCount; i++ {
// 		go worker(repo)
// 	}
// }

// //worker - logic
// //claim the payment
// //create the payment intent by calling the Extrnal Psp -- will return the client_secreat key
// //presist payment status to "Processing"
// // send the client_secreat to the client
// // after the the client conformation the webhook will trigger

// func worker(repo workerRepository) {
// 	for {
// 		//claim the payment
// 		paymentDetails, err := repo.ClaimPayment()
// 		if err == sql.ErrNoRows {
// 			//sleep fop 2 seconds
// 			//log.Println("no work available")
// 			time.Sleep(time.Second * 5)
// 			continue
// 		}
// 		if err != nil {
// 			log.Println("claim error:", err)
// 			time.Sleep(2 * time.Second)
// 			continue
// 		}
// 		//log.Println(paymentDetails)
// 		//call the External PSP
// 		pspRefID, err := stripeclient.CreatePaymentIntent(paymentDetails)
// 		if err != nil {
// 			log.Printf("Error in Worker calling External PSP: %s", err)
// 			continue
// 		}
// 		log.Println(pspRefID)
// 		//update the payment_intent State
// 		if err := repo.MarkProcessing(paymentDetails.PaymentId, pspRefID); err != nil {
// 			log.Printf("MarkProcessing error: %v", err)
// 		}

// 		//retun the client_secret top the client
// 	}
// }
