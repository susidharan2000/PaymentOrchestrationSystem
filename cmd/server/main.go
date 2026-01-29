package main

import (
	"fmt"
	"log"
	"net/http"

	internaldb "github.com/susidharan/payment-orchestration-system/internal/database"
	internalhttp "github.com/susidharan/payment-orchestration-system/internal/http"
	"github.com/susidharan/payment-orchestration-system/internal/payment/intent"
)

func main() {
	db := internaldb.New()
	defer db.Close()

	// Create tables
	intent.CreatePaymentIntentTable(db)

	router := internalhttp.NewRouter(db)
	port := 8080
	adr := fmt.Sprintf(":%v", port)
	srv := &http.Server{
		Addr:    adr,
		Handler: router,
	}
	// start server
	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Println("http server error:", err)
	}
}
