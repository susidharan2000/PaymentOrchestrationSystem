package main

import (
	"fmt"
	"log"
	"net/http"

	"github.com/joho/godotenv"
	internaldb "github.com/susidharan/payment-orchestration-system/internal/database"
	internalhttp "github.com/susidharan/payment-orchestration-system/internal/http"
	linker "github.com/susidharan/payment-orchestration-system/internal/linker"
	paymentIntent "github.com/susidharan/payment-orchestration-system/internal/payment/intent"
	stripeclient "github.com/susidharan/payment-orchestration-system/internal/psp/stripe"
	Webhook_ingestor "github.com/susidharan/payment-orchestration-system/internal/webhook_ingestor"
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
	// get Worker Repo
	workerRepo := worker.NewWorkerRepository(db)
	//web hook Repo
	webhookRepo := Webhook_ingestor.NewWebhookRepository(db)
	//linker Repo
	linkerRepo := linker.NewLinkerRepository(db)

	go worker.StartWorkers(workerRepo) //start payment_Worker poll

	go linker.StartLinker(linkerRepo)

	router := internalhttp.NewRouter(paymentRepo, webhookRepo)
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
