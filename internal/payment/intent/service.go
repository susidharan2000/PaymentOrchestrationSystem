package intent

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"
)

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
	if err := req.validateRequest(); err != nil {
		ErrorResponse(w, http.StatusBadRequest, err.Error())
		return
	}
	// Create the payment Request
	id, created, err := repo.PersistPaymentRequest(req)
	if err != nil {
		log.Print(err)
		ErrorResponse(w, http.StatusServiceUnavailable, "temporary server error, try again later")
		return
	}
	SuccessResponse(w, id, req, created)
	//fmt.Println(responce)
}

//response Writter

func ErrorResponse(w http.ResponseWriter, s int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(s)
	json.NewEncoder(w).Encode(map[string]string{
		"message": message,
	})
}
func SuccessResponse(w http.ResponseWriter, id string, req CreatePaymentRequest, created bool) {
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
	responce.Status = "CREATED"
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
