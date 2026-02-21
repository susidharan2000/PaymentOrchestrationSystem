package stateprojector

import (
	"log"
	"time"
)

func StartProjector(repo ProjectorRepository) {
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()
	for range ticker.C {
		if err := repo.projectState(); err != nil {
			log.Println(err)
		}
	}
}
