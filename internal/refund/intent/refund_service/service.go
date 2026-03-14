package refundservice

import (
	"encoding/json"
	"net/http"

	chi "github.com/go-chi/chi/v5"
	refund_model "github.com/susidharan/payment-orchestration-system/internal/refund/intent/model"
	refund_repo "github.com/susidharan/payment-orchestration-system/internal/refund/intent/refund_repository"
)

// in refund - getting the payment_id(internl) in header
//
//		payload = {
//		"amount":1000,
//	    "idempotency_key":"sample_idempotency_key_123"
//		}
//
// 1: validate the refund request - query the payment_intent_table check the (captured_amount - refunded_amount >= refund_amount) if true then return paymentDetails
// 2: After validation create the refund Payment in Refund Record
func CreateRefundIntent(w http.ResponseWriter, r *http.Request, repo refund_repo.RefundRepository) {
	defer r.Body.Close()
	paymentID := chi.URLParam(r, "id")
	var req refund_model.RefundRequest
	w.Header().Set("Content-Type", "application/json")
	//validate the Request
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		ErrorResponse(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.Amount <= 0 || req.Idempotencykey == "" {
		ErrorResponse(w, http.StatusBadRequest, "invalid Payload ")
		return
	}
	//create the refund
	refundDetails, ok, err := repo.CreateRefundRecord(req, paymentID)
	if err != nil {
		switch err.Error() {
		case "payment_not_refundable":
			ErrorResponse(w, http.StatusConflict, err.Error())
		default:
			ErrorResponse(w, http.StatusInternalServerError, err.Error())
		}
		return
	}
	if !ok {
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(refundDetails)
		return
	}

	//success responce
	w.WriteHeader(http.StatusAccepted)
	json.NewEncoder(w).Encode(refundDetails)

}

// response Writter
func ErrorResponse(w http.ResponseWriter, s int, message string) {
	w.WriteHeader(s)
	json.NewEncoder(w).Encode(map[string]string{
		"error": message,
	})
}
