package storage

import (
	"context"
	"log"
	"time"
)

// CleanupWorker handles periodic cleanup of temporary files
type CleanupWorker struct {
	storage   *s3Storage
	interval  time.Duration
	stopCh    chan struct{}
	isRunning bool
}

// NewCleanupWorker creates a new cleanup worker
func NewCleanupWorker(storage *s3Storage, interval time.Duration) *CleanupWorker {
	return &CleanupWorker{
		storage:  storage,
		interval: interval,
		stopCh:   make(chan struct{}),
	}
}

// Start begins the periodic cleanup process
func (w *CleanupWorker) Start() {
	if w.isRunning {
		return
	}
	w.isRunning = true

	go func() {
		ticker := time.NewTicker(w.interval)
		defer ticker.Stop()

		// Run initial cleanup
		if err := w.storage.CleanupTempFiles(context.Background()); err != nil {
			log.Printf("Error during initial cleanup: %v", err)
		}

		for {
			select {
			case <-ticker.C:
				if err := w.storage.CleanupTempFiles(context.Background()); err != nil {
					log.Printf("Error during cleanup: %v", err)
				}
			case <-w.stopCh:
				return
			}
		}
	}()
}

// Stop halts the cleanup process
func (w *CleanupWorker) Stop() {
	if !w.isRunning {
		return
	}
	w.isRunning = false
	close(w.stopCh)
}
