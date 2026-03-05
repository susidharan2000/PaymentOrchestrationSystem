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

	"github.com/go-chi/chi/v5"
	model "github.com/susidharan/payment-orchestration-system/internal/payment/intent/model"
	Repository "github.com/susidharan/payment-orchestration-system/internal/payment/intent/payment_repository"
	stripe_handler "github.com/susidharan/payment-orchestration-system/internal/payment/intent/stripe_handler"
)

// create the payment
func CreatePayment(w http.ResponseWriter, r *http.Request, repo Repository.PaymentRepository) {

	bodyBytes, err := io.ReadAll(r.Body)
	defer r.Body.Close()
	if err != nil {
		ErrorResponse(w, http.StatusBadRequest, err.Error())
		return
	}
	var req model.CreatePaymentRequest
	jsonData := json.NewDecoder(bytes.NewReader(bodyBytes))
	jsonData.DisallowUnknownFields()
	if err := jsonData.Decode(&req); err != nil {
		ErrorResponse(w, http.StatusBadRequest, err.Error())
		return
	}
	//validate the request
	if err := validateRequest(req); err != nil {
		ErrorResponse(w, http.StatusBadRequest, err.Error())
		return
	}
	//hash Request
	var paymentHash model.PaymentFingerprint
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
	switch req.PspName {
	case "stripe":
		stripe_handler.HandleStripePayment(w, paymentDetails, created, repo)
	}
}

// get payment
func GetPaymentDetails(w http.ResponseWriter, r *http.Request, repo Repository.PaymentRepository) {
	paymentID := chi.URLParam(r, "id")
	PaymentDetails, err := repo.GetPaymentById(paymentID)
	if err != nil {
		ErrorResponse(w, http.StatusInternalServerError, "Internal Server Error")
		return
	}
	//return success response
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

// validate the request
func validateRequest(r model.CreatePaymentRequest) error {
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

func ComputeRequestHash(f model.PaymentFingerprint) (string, error) {
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
