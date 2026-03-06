package stripehandler

import (
	"encoding/json"
	"log"
	"net/http"
	"os"

	"github.com/susidharan/payment-orchestration-system/internal/domain"
	model "github.com/susidharan/payment-orchestration-system/internal/payment/intent/model"
	Repository "github.com/susidharan/payment-orchestration-system/internal/payment/intent/payment_repository"
	"github.com/susidharan/payment-orchestration-system/internal/psp"
)

func HandleStripePayment(w http.ResponseWriter, paymentDetails domain.PaymentParams, created bool, repo Repository.PaymentRepository, registry *psp.Registry) {
	var pspRefID string
	var client_secret string
	var err error
	pspProvider, ok := registry.Get(paymentDetails.PspName)
	if !ok {
		ErrorResponse(w, http.StatusBadRequest, "invalid PSP")
		return
	}
	if created {
		//log.Print("New Payment Created")
		pspRefID, client_secret, err = pspProvider.CreatePaymentIntent(paymentDetails)
		if err != nil {
			//log.Printf("Error in calling External PSP: %s", err)
			ErrorResponse(w, http.StatusBadGateway, "psp Error")
			return
		}
	} else {
		//log.Println(paymentDetails.PspRefID)
		if !paymentDetails.PspRefID.Valid {
			//log.Print("New Payment Created")
			pspRefID, client_secret, err = pspProvider.CreatePaymentIntent(paymentDetails)
			if err != nil {
				//log.Printf("Error in calling external PSP: %s", err)
				ErrorResponse(w, http.StatusBadGateway, "psp Error")
				return
			}
		} else {
			//retry
			log.Print("retry Happened")
			pi, err := pspProvider.GetPaymentIntent(paymentDetails.PspRefID.String)
			if err != nil {
				//log.Printf("Error in calling External PSP: %s", err)
				ErrorResponse(w, http.StatusBadGateway, "psp Error")
				return
			}
			SuccessResponse(w, paymentDetails.PaymentId, paymentDetails, created, pi.ClientSecret, "PROCESSING")
			return
		}
	}
	//log.Printf("PSP_REFERANCE ID :%s", pspRefID)
	//log.Printf("PSP_CLIENT : %s", client_secret)
	//presist payment status to Processing , presist psp_ref_id
	if err := repo.MarkProcessing(paymentDetails.PaymentId, pspRefID); err != nil {
		log.Printf("MarkProcessing error: %v", err)
	}
	//return client_secret Key in respince
	SuccessResponse(w, paymentDetails.PaymentId, paymentDetails, created, client_secret, "PROCESSING")
}

// response Writter
func ErrorResponse(w http.ResponseWriter, s int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(s)
	json.NewEncoder(w).Encode(map[string]string{
		"message": message,
	})
}
func SuccessResponse(w http.ResponseWriter, id string, paymentDetails domain.PaymentParams, created bool, client_secret string, status string) {
	w.Header().Set("Content-Type", "application/json")
	if created {
		w.WriteHeader(http.StatusCreated)
	} else {
		w.WriteHeader(http.StatusOK)
	}
	var responce model.PaymentDetails
	responce.PaymentId = id
	responce.Amount = paymentDetails.Amount
	responce.Currency = paymentDetails.Currency
	responce.PspName = paymentDetails.PspName
	responce.Status = status
	responce.ClientToken = client_secret
	stripePublishablekey := os.Getenv("STRIPE_PUBLISHABLE_KEY")
	responce.Publishablekey = stripePublishablekey
	json.NewEncoder(w).Encode(responce)
}
