package main

import (
	"fmt"
	"log"
	"net/http"

	"github.com/joho/godotenv"
	internaldb "github.com/susidharan/payment-orchestration-system/internal/database"
	internalhttp "github.com/susidharan/payment-orchestration-system/internal/http"
	paymentIntent "github.com/susidharan/payment-orchestration-system/internal/payment/intent"
	stripeclient "github.com/susidharan/payment-orchestration-system/internal/psp/stripe"
	worker "github.com/susidharan/payment-orchestration-system/internal/worker"
)

func main() {
	db := internaldb.New()
	defer db.Close()

	if err := godotenv.Load(); err != nil {
		log.Print(err)
	}

	// PSP Init's
	stripeclient.Init()

	// get payment Repo
	paymentRepo := paymentIntent.NewPaymentRepository(db)
	err := paymentRepo.CreatePaymentIntentTable() // Create tables
	if err != nil {
		log.Fatal("Table creation Failed")
	}

	// get Worker Repo
	workerRepo := worker.NewWorkerRepository(db)
	//start paymentWorker poll
	go worker.StartWorkers(workerRepo)

	router := internalhttp.NewRouter(paymentRepo)
	port := 8080
	adr := fmt.Sprintf(":%v", port)
	srv := &http.Server{
		Addr:    adr,
		Handler: router,
	}
	// start server
	if err = srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Println("http server error:", err)
	}
}
