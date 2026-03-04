package intent

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"
	"os"

	"github.com/go-chi/chi/v5"
	stripeclient "github.com/susidharan/payment-orchestration-system/internal/psp/stripe"
)

// create the payment
func CreatePayment(w http.ResponseWriter, r *http.Request, repo PaymentRepository) {

	bodyBytes, err := io.ReadAll(r.Body)
	defer r.Body.Close()
	if err != nil {
		ErrorResponse(w, http.StatusBadRequest, err.Error())
		return
	}
	var req CreatePaymentRequest
	jsonData := json.NewDecoder(bytes.NewReader(bodyBytes))
	jsonData.DisallowUnknownFields()
	if err := jsonData.Decode(&req); err != nil {
		ErrorResponse(w, http.StatusBadRequest, err.Error())
		return
	}
	//validate the request
	if err := req.validateRequest(); err != nil {
		ErrorResponse(w, http.StatusBadRequest, err.Error())
		return
	}
	//hash Request
	var paymentHash PaymentFingerprint
	paymentHash.Amount = req.Amount
	paymentHash.Currency = req.Currency
	paymentHash.PSPName = req.PspName
	requestHash, err := ComputeRequestHash(paymentHash)
	if err != nil {
		message := "HashCode Generation failed"
		log.Println(message)
		ErrorResponse(w, http.StatusInternalServerError, message)
		return
	}
	// Create the payment Request
	paymentDetails, created, err := repo.PersistPaymentRequest(req, requestHash)
	if err != nil {
		log.Print(err)
		switch err.Error() {
		case "idempotency key reused with different payload":
			ErrorResponse(w, http.StatusConflict, err.Error())
		default:
			ErrorResponse(w, http.StatusInternalServerError, err.Error())
		}
		return
	}
	//log.Printf("Payment Details:%v", paymentDetails)
	//call the external PSP
	var pspRefID string
	var client_secret string
	if created {
		//log.Print("New Payment Created")
		pspRefID, client_secret, err = stripeclient.CreatePaymentIntent(paymentDetails)
		if err != nil {
			//log.Printf("Error in calling External PSP: %s", err)
			ErrorResponse(w, http.StatusBadGateway, "psp Error")
			return
		}
	} else {
		//log.Println(paymentDetails.PspRefID)
		if !paymentDetails.PspRefID.Valid {
			//log.Print("New Payment Created")
			pspRefID, client_secret, err = stripeclient.CreatePaymentIntent(paymentDetails)
			if err != nil {
				//log.Printf("Error in calling External PSP: %s", err)
				ErrorResponse(w, http.StatusBadGateway, "psp Error")
				return
			}
		} else {
			//retry
			log.Print("retry Happened")
			pi, err := stripeclient.GetPaymentIntent(paymentDetails.PspRefID.String)
			if err != nil {
				//log.Printf("Error in calling External PSP: %s", err)
				ErrorResponse(w, http.StatusBadGateway, "psp Error")
				return
			}
			SuccessResponse(w, paymentDetails.PaymentId, req, created, pi.ClientSecret, "PROCESSING")
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
	SuccessResponse(w, paymentDetails.PaymentId, req, created, client_secret, "PROCESSING")
}

// get payment
func GetPaymentDetails(w http.ResponseWriter, r *http.Request, repo PaymentRepository) {
	paymentID := chi.URLParam(r, "id")
	PaymentDetails, err := repo.getPaymentById(paymentID)
	if err != nil {
		ErrorResponse(w, http.StatusInternalServerError, "Internal Server Error")
		return
	}
	//return success responc
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(PaymentDetails)

}

// response Writter
func ErrorResponse(w http.ResponseWriter, s int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(s)
	json.NewEncoder(w).Encode(map[string]string{
		"message": message,
	})
}
func SuccessResponse(w http.ResponseWriter, id string, req CreatePaymentRequest, created bool, client_secret string, status string) {
	w.Header().Set("Content-Type", "application/json")
	if created {
		w.WriteHeader(http.StatusCreated)
	} else {
		w.WriteHeader(http.StatusOK)
	}
	var responce PaymentDetails
	responce.PaymentId = id
	responce.Amount = req.Amount
	responce.Currency = req.Currency
	responce.PspName = req.PspName
	responce.Status = status
	responce.ClientSecret = client_secret
	stripePublishablekey := os.Getenv("STRIPE_PUBLISHABLE_KEY")
	responce.Publishablekey = stripePublishablekey
	json.NewEncoder(w).Encode(responce)
}

// validate the request
func (r CreatePaymentRequest) validateRequest() error {
	if r.Amount <= 0 {
		return errors.New("invalid amount")
	}
	if len(r.Currency) != 3 {
		return errors.New("currency must be 3 letters")
	}
	if r.PspName == "" {
		return errors.New("psp_name is required")
	}
	if r.IdempotencyKey == "" {
		return errors.New("idempotency_key is required")
	}
	return nil
}

func ComputeRequestHash(f PaymentFingerprint) (string, error) {
	b, err := json.Marshal(f)
	if err != nil {
		return "", err
	}

	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:]), nil
}

// func CancelPayment(w http.ResponseWriter, r *http.Request, repo PaymentRepository) {
// 	//cancel the payment only if the external psp call not is made because
// 	//canceling the payment after the not possible and we handle that in refund

// 	paymentID := chi.URLParam(r, "id")
// 	//log.Println(paymentID)
// 	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second) //set timeout for the request
// 	defer cancel()

// 	err := repo.CancelPayment(ctx, paymentID)
// 	if err != nil {
// 		http.Error(w, err.Error(), http.StatusConflict)
// 		return
// 	}

// 	w.WriteHeader(http.StatusOK)
// 	w.Header().Set("Content-Type", "application/json")
// 	json.NewEncoder(w).Encode(map[string]string{
// 		"Payment ID": paymentID,
// 		"message":    "Payment Cancelled",
// 	})
// }
