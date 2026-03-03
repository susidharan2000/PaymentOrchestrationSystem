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
	state_projector "github.com/susidharan/payment-orchestration-system/internal/state_projector"
	Webhook_ingestor "github.com/susidharan/payment-orchestration-system/internal/webhook_ingestor"
)

func main() {
	db := internaldb.New()
	defer db.Close()

	if err := godotenv.Load(); err != nil {
		log.Print(err)
	}

	//seed the Jitter
	//r := rand.New(rand.NewSource(time.Now().UnixNano()))

	// PSP Init's
	stripeclient.Init()

	// get payment Repo
	paymentRepo := paymentIntent.NewPaymentRepository(db)
	// get Worker Repo
	//workerRepo := worker.NewWorkerRepository(db)
	//web hook Repo
	webhookRepo := Webhook_ingestor.NewWebhookRepository(db)
	//Projector Repo
	projectorRepo := state_projector.NewProjectorRepository(db)
	//Reconciler Repository
	//reconcilerRepo := reconciler.NewReconcilerRepository(db)

	//go worker.StartWorkers(workerRepo) //start payment_Worker poll

	go state_projector.StartProjector(projectorRepo) // start State Projector

	//go reconciler.StartReconciler(reconcilerRepo, r) // start Reconciler

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
