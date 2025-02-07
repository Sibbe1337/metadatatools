package domain

import (
	"context"
	"encoding/json"
	"time"
)

// JobStatus represents the current state of a job
type JobStatus string

const (
	JobStatusPending    JobStatus = "pending"
	JobStatusProcessing JobStatus = "processing"
	JobStatusCompleted  JobStatus = "completed"
	JobStatusFailed     JobStatus = "failed"
	JobStatusCanceled   JobStatus = "canceled"
)

// JobPriority represents the priority level of a job
type JobPriority int

const (
	JobPriorityLow    JobPriority = 0
	JobPriorityNormal JobPriority = 1
	JobPriorityHigh   JobPriority = 2
)

// JobType represents the type of job
type JobType string

const (
	JobTypeAudioProcess JobType = "audio_process"
	JobTypeAIEnrich     JobType = "ai_enrich"
	JobTypeDDEXExport   JobType = "ddex_export"
	JobTypeCleanup      JobType = "cleanup"
)

// Job represents a background job
type Job struct {
	ID          string          `json:"id"`
	Type        JobType         `json:"type"`
	Priority    JobPriority     `json:"priority"`
	Status      JobStatus       `json:"status"`
	Payload     json.RawMessage `json:"payload"`
	Error       string          `json:"error,omitempty"`
	Progress    int             `json:"progress"`
	RetryCount  int             `json:"retry_count"`
	MaxRetries  int             `json:"max_retries"`
	CreatedAt   time.Time       `json:"created_at"`
	StartedAt   *time.Time      `json:"started_at,omitempty"`
	CompletedAt *time.Time      `json:"completed_at,omitempty"`
	NextRetryAt *time.Time      `json:"next_retry_at,omitempty"`
}

// JobConfig holds configuration for the job system
type JobConfig struct {
	// Worker settings
	NumWorkers    int           `env:"JOB_NUM_WORKERS" envDefault:"5"`
	MaxConcurrent int           `env:"JOB_MAX_CONCURRENT" envDefault:"10"`
	PollInterval  time.Duration `env:"JOB_POLL_INTERVAL" envDefault:"1s"`
	ShutdownWait  time.Duration `env:"JOB_SHUTDOWN_WAIT" envDefault:"30s"`

	// Job settings
	DefaultMaxRetries int           `env:"JOB_DEFAULT_MAX_RETRIES" envDefault:"3"`
	DefaultTTL        time.Duration `env:"JOB_DEFAULT_TTL" envDefault:"24h"`
	MaxPayloadSize    int64         `env:"JOB_MAX_PAYLOAD_SIZE" envDefault:"1048576"` // 1MB

	// Queue settings
	QueuePrefix     string        `env:"JOB_QUEUE_PREFIX" envDefault:"jobs:"`
	RetryDelay      time.Duration `env:"JOB_RETRY_DELAY" envDefault:"5s"`
	MaxRetryDelay   time.Duration `env:"JOB_MAX_RETRY_DELAY" envDefault:"1h"`
	RetryMultiplier float64       `env:"JOB_RETRY_MULTIPLIER" envDefault:"2.0"`

	// Cleanup settings
	CleanupInterval time.Duration `env:"JOB_CLEANUP_INTERVAL" envDefault:"1h"`
	MaxJobAge       time.Duration `env:"JOB_MAX_AGE" envDefault:"168h"` // 7 days
}

// JobQueue defines the interface for job queue operations
type JobQueue interface {
	// Enqueue adds a new job to the queue
	Enqueue(ctx context.Context, job *Job) error

	// Dequeue gets the next job to process
	Dequeue(ctx context.Context) (*Job, error)

	// Complete marks a job as completed
	Complete(ctx context.Context, jobID string) error

	// Fail marks a job as failed
	Fail(ctx context.Context, jobID string, err error) error

	// Cancel cancels a pending or running job
	Cancel(ctx context.Context, jobID string) error

	// GetStatus gets the current status of a job
	GetStatus(ctx context.Context, jobID string) (*Job, error)

	// UpdateProgress updates the progress of a running job
	UpdateProgress(ctx context.Context, jobID string, progress int) error
}

// JobHandler defines the interface for job type handlers
type JobHandler interface {
	// HandleJob processes a specific type of job
	HandleJob(ctx context.Context, job *Job) error

	// JobType returns the type of job this handler processes
	JobType() JobType
}

// JobProcessor manages the job processing system
type JobProcessor interface {
	// Start starts the job processor
	Start(ctx context.Context) error

	// Stop stops the job processor
	Stop() error

	// RegisterHandler registers a handler for a specific job type
	RegisterHandler(handler JobHandler) error
}
