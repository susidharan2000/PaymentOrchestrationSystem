package linker

import (
	"log"
	"os"
	"strconv"
	"time"
)

// link the ledger with payment_intent via psp_ref_id
func StartLinker(repo LinkerRepository) {
	LinkerWorkerCount := 2

	if v := os.Getenv("LINKER_COUNT"); v != "" {
		if parsed, err := strconv.Atoi(v); err == nil && parsed > 0 {
			LinkerWorkerCount = parsed
		}
	}
	// inisiate workers
	for i := 0; i < LinkerWorkerCount; i++ {
		go worker(repo)
	}
}

// worker
func worker(repo LinkerRepository) {
	for {
		rows, err := repo.linkLedger()
		if err != nil {
			log.Println(err)
			time.Sleep(2 * time.Second)
			continue
		}
		if err != nil {
		}

		if rows == 0 {
			time.Sleep(2 * time.Second)
			continue
		}
	}
}
