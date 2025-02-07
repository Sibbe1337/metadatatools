package jobs

import (
	"context"
	"fmt"
	"sync"
	"time"

	"metadatatool/internal/pkg/domain"
	"metadatatool/internal/pkg/metrics"
)

// Processor implements domain.JobProcessor
type Processor struct {
	queue    domain.JobQueue
	config   *domain.JobConfig
	handlers map[domain.JobType]domain.JobHandler
	workers  []*worker
	wg       sync.WaitGroup
	mu       sync.RWMutex
	ctx      context.Context
	cancel   context.CancelFunc
}

// worker represents a job processing worker
type worker struct {
	id       int
	queue    domain.JobQueue
	handlers map[domain.JobType]domain.JobHandler
	wg       *sync.WaitGroup
}

// NewProcessor creates a new job processor
func NewProcessor(queue domain.JobQueue, config *domain.JobConfig) *Processor {
	ctx, cancel := context.WithCancel(context.Background())
	return &Processor{
		queue:    queue,
		config:   config,
		handlers: make(map[domain.JobType]domain.JobHandler),
		ctx:      ctx,
		cancel:   cancel,
	}
}

// RegisterHandler registers a handler for a specific job type
func (p *Processor) RegisterHandler(handler domain.JobHandler) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	jobType := handler.JobType()
	if _, exists := p.handlers[jobType]; exists {
		return fmt.Errorf("handler already registered for job type: %s", jobType)
	}

	p.handlers[jobType] = handler
	return nil
}

// Start starts the job processor
func (p *Processor) Start(ctx context.Context) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if len(p.handlers) == 0 {
		return fmt.Errorf("no job handlers registered")
	}

	// Create worker pool
	p.workers = make([]*worker, p.config.NumWorkers)
	for i := 0; i < p.config.NumWorkers; i++ {
		w := &worker{
			id:       i,
			queue:    p.queue,
			handlers: p.handlers,
			wg:       &p.wg,
		}
		p.workers[i] = w
		p.wg.Add(1)
		go w.start(p.ctx)
	}

	return nil
}

// Stop stops the job processor
func (p *Processor) Stop() error {
	p.cancel()
	done := make(chan struct{})
	go func() {
		p.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		return nil
	case <-time.After(p.config.ShutdownWait):
		return fmt.Errorf("shutdown timed out after %v", p.config.ShutdownWait)
	}
}

// start starts the worker's processing loop
func (w *worker) start(ctx context.Context) {
	defer w.wg.Done()

	for {
		select {
		case <-ctx.Done():
			return
		default:
			if err := w.processNextJob(ctx); err != nil {
				// Log error but continue processing
				metrics.JobErrors.WithLabelValues("worker", "process_error").Inc()
			}
		}
	}
}

// processNextJob processes the next available job
func (w *worker) processNextJob(ctx context.Context) error {
	// Dequeue job
	job, err := w.queue.Dequeue(ctx)
	if err != nil {
		return fmt.Errorf("failed to dequeue job: %w", err)
	}

	// No job available
	if job == nil {
		time.Sleep(100 * time.Millisecond) // Prevent tight loop
		return nil
	}

	// Get handler for job type
	handler, ok := w.handlers[job.Type]
	if !ok {
		err := fmt.Errorf("no handler registered for job type: %s", job.Type)
		if err := w.queue.Fail(ctx, job.ID, err); err != nil {
			return fmt.Errorf("failed to mark job as failed: %w", err)
		}
		return err
	}

	// Process job
	if err := handler.HandleJob(ctx, job); err != nil {
		if err := w.queue.Fail(ctx, job.ID, err); err != nil {
			return fmt.Errorf("failed to mark job as failed: %w", err)
		}
		return err
	}

	// Mark job as completed
	if err := w.queue.Complete(ctx, job.ID); err != nil {
		return fmt.Errorf("failed to mark job as completed: %w", err)
	}

	return nil
}
