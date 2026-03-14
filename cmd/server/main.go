package main

import (
	"fmt"
	"log"
	"net/http"

	"github.com/joho/godotenv"
	internaldb "github.com/susidharan/payment-orchestration-system/internal/database"
	internalhttp "github.com/susidharan/payment-orchestration-system/internal/http"
	paymentrepo "github.com/susidharan/payment-orchestration-system/internal/payment/intent/payment_repository"
	psp "github.com/susidharan/payment-orchestration-system/internal/psp"
	stripePSP "github.com/susidharan/payment-orchestration-system/internal/psp/stripe"

	//"github.com/susidharan/payment-orchestration-system/internal/reconciler"
	refund_repo "github.com/susidharan/payment-orchestration-system/internal/refund/intent/refund_repository"
	state_projector "github.com/susidharan/payment-orchestration-system/internal/state_projector"
	Webhook_Repo "github.com/susidharan/payment-orchestration-system/internal/webhook_ingestor/webhook_repository"
)

func main() {
	//load env
	if err := godotenv.Load(); err != nil {
		log.Print(err)
	}

	//open DB connection
	db := internaldb.New()
	defer db.Close()

	//seed the Jitter
	//r := rand.New(rand.NewSource(time.Now().UnixNano()))

	// Psp registry
	registry := psp.NewRegistry()
	registry.Register("stripe", stripePSP.NewAdapter())

	// get payment Repo
	paymentRepo := paymentrepo.NewPaymentRepository(db)
	// get Worker Repo
	//workerRepo := worker.NewWorkerRepository(db)
	//web hook Repo
	webhookRepo := Webhook_Repo.NewWebhookRepository(db)
	//Projector Repo
	projectorRepo := state_projector.NewProjectorRepository(db)
	//Reconciler Repository
	//reconcilerRepo := reconciler.NewReconcilerRepository(db)
	//refund Intent Repo
	refundIntentRepo := refund_repo.NewRefundRepository(db)

	go state_projector.StartProjector(projectorRepo) // start State Projector

	//go reconciler.StartReconciler(reconcilerRepo, r, registry) // start Reconciler

	//go worker.StartRefundWorkers(workerRepo) //start Refund_Worker poll

	router := internalhttp.NewRouter(paymentRepo, webhookRepo, refundIntentRepo, registry)
	port := 8080
	adr := fmt.Sprintf(":%v", port)
	srv := &http.Server{
		Addr:    adr,
		Handler: cors(router),
	}
	// start server
	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Println("http server error:", err)
	}
}

// CORS
func cors(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		// Important for preflight
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}
