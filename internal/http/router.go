package http

import (
	"database/sql"
	"net/http"

	intentService "github.com/susidharan/payment-orchestration-system/internal/payment/intent"
)

func NewRouter(db *sql.DB) http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("/payments", func(w http.ResponseWriter, r *http.Request) {
		intentService.CreatePayment(w, r, db)
	})

	return mux
}
