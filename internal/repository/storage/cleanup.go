package storage

import (
	"context"
	"log"
	"time"
)

// StartCleanupWorker starts a goroutine that periodically cleans up temporary files
func (s *s3Storage) StartCleanupWorker(ctx context.Context) {
	if s.cfg.CleanupInterval <= 0 {
		log.Printf("Cleanup worker disabled: cleanup interval is %v", s.cfg.CleanupInterval)
		return
	}

	go func() {
		ticker := time.NewTicker(s.cfg.CleanupInterval)
		defer ticker.Stop()

		// Run initial cleanup
		if err := s.CleanupTempFiles(ctx); err != nil {
			log.Printf("Error during initial cleanup: %v", err)
		}

		for {
			select {
			case <-ticker.C:
				if err := s.CleanupTempFiles(ctx); err != nil {
					log.Printf("Error during cleanup: %v", err)
				}
			case <-ctx.Done():
				log.Printf("Cleanup worker stopped: %v", ctx.Err())
				return
			}
		}
	}()

	log.Printf("Cleanup worker started with interval %v", s.cfg.CleanupInterval)
}
