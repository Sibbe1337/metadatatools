package jobs

import (
	"context"
	"encoding/json"
	"fmt"
	"metadatatool/internal/pkg/domain"
	"metadatatool/internal/pkg/metrics"
	"time"

	"github.com/redis/go-redis/v9"
)

const (
	// Queue keys
	queueKeyPrefix     = "jobs:"
	queueKeyPending    = "pending"
	queueKeyProcessing = "processing"
	queueKeyCompleted  = "completed"
	queueKeyFailed     = "failed"

	// Hash fields
	hashFieldJob        = "job"
	hashFieldStatus     = "status"
	hashFieldProgress   = "progress"
	hashFieldError      = "error"
	hashFieldStartedAt  = "started_at"
	hashFieldRetryCount = "retry_count"
)

// RedisQueue implements domain.JobQueue using Redis
type RedisQueue struct {
	client *redis.Client
	config *domain.JobConfig
}

// NewRedisQueue creates a new Redis-backed job queue
func NewRedisQueue(client *redis.Client, config *domain.JobConfig) *RedisQueue {
	return &RedisQueue{
		client: client,
		config: config,
	}
}

// queueKey returns the Redis key for a specific queue
func (q *RedisQueue) queueKey(queueType string) string {
	return fmt.Sprintf("%s%s", q.config.QueuePrefix, queueType)
}

// jobKey returns the Redis key for a specific job
func (q *RedisQueue) jobKey(jobID string) string {
	return fmt.Sprintf("%s:job:%s", q.config.QueuePrefix, jobID)
}

// Enqueue adds a new job to the queue
func (q *RedisQueue) Enqueue(ctx context.Context, job *domain.Job) error {
	// Start pipeline
	pipe := q.client.Pipeline()

	// Serialize job
	jobData, err := json.Marshal(job)
	if err != nil {
		return fmt.Errorf("failed to marshal job: %w", err)
	}

	// Store job data in hash
	jobKey := q.jobKey(job.ID)
	pipe.HSet(ctx, jobKey, map[string]interface{}{
		hashFieldJob:    string(jobData),
		hashFieldStatus: string(job.Status),
	})
	pipe.Expire(ctx, jobKey, q.config.DefaultTTL)

	// Add to pending queue with priority score
	score := float64(time.Now().UnixNano()) - float64(job.Priority)*1e12 // Lower score = higher priority
	pipe.ZAdd(ctx, q.queueKey(queueKeyPending), redis.Z{
		Score:  score,
		Member: job.ID,
	})

	// Execute pipeline
	if _, err := pipe.Exec(ctx); err != nil {
		return fmt.Errorf("failed to enqueue job: %w", err)
	}

	// Record metrics
	metrics.JobsInQueue.WithLabelValues(string(job.Type)).Inc()
	metrics.JobStatusTransitions.WithLabelValues(string(job.Type), "", string(domain.JobStatusPending)).Inc()

	return nil
}

// Dequeue gets the next job to process
func (q *RedisQueue) Dequeue(ctx context.Context) (*domain.Job, error) {
	// Start pipeline
	pipe := q.client.Pipeline()

	// Get highest priority job from pending queue
	pendingKey := q.queueKey(queueKeyPending)
	processingKey := q.queueKey(queueKeyProcessing)
	result := pipe.ZPopMin(ctx, pendingKey, 1)

	// Execute pipeline to get job ID
	if _, err := pipe.Exec(ctx); err != nil {
		return nil, fmt.Errorf("failed to dequeue job: %w", err)
	}

	// Check if queue is empty
	popResult := result.Val()
	if len(popResult) == 0 {
		return nil, nil
	}

	jobID := popResult[0].Member.(string)
	jobKey := q.jobKey(jobID)

	// Get job data
	jobData, err := q.client.HGet(ctx, jobKey, hashFieldJob).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to get job data: %w", err)
	}

	// Deserialize job
	var job domain.Job
	if err := json.Unmarshal([]byte(jobData), &job); err != nil {
		return nil, fmt.Errorf("failed to unmarshal job: %w", err)
	}

	// Update job status
	now := time.Now()
	job.Status = domain.JobStatusProcessing
	job.StartedAt = &now

	// Update job in Redis
	pipe = q.client.Pipeline()
	updatedJobData, _ := json.Marshal(job)
	pipe.HSet(ctx, jobKey, map[string]interface{}{
		hashFieldJob:       string(updatedJobData),
		hashFieldStatus:    string(job.Status),
		hashFieldStartedAt: now.Format(time.RFC3339),
	})
	pipe.ZAdd(ctx, processingKey, redis.Z{
		Score:  float64(now.UnixNano()),
		Member: jobID,
	})

	// Execute pipeline
	if _, err := pipe.Exec(ctx); err != nil {
		return nil, fmt.Errorf("failed to update job status: %w", err)
	}

	// Record metrics
	metrics.JobsInQueue.WithLabelValues(string(job.Type)).Dec()
	metrics.JobStatusTransitions.WithLabelValues(string(job.Type), string(domain.JobStatusPending), string(domain.JobStatusProcessing)).Inc()
	metrics.JobQueueLatency.WithLabelValues(string(job.Type)).Observe(time.Since(job.CreatedAt).Seconds())

	return &job, nil
}

// Complete marks a job as completed
func (q *RedisQueue) Complete(ctx context.Context, jobID string) error {
	jobKey := q.jobKey(jobID)
	processingKey := q.queueKey(queueKeyProcessing)
	completedKey := q.queueKey(queueKeyCompleted)

	// Get current job data
	jobData, err := q.client.HGet(ctx, jobKey, hashFieldJob).Result()
	if err != nil {
		return fmt.Errorf("failed to get job data: %w", err)
	}

	// Deserialize job
	var job domain.Job
	if err := json.Unmarshal([]byte(jobData), &job); err != nil {
		return fmt.Errorf("failed to unmarshal job: %w", err)
	}

	// Update job status
	now := time.Now()
	prevStatus := job.Status
	job.Status = domain.JobStatusCompleted
	job.CompletedAt = &now

	// Start pipeline
	pipe := q.client.Pipeline()

	// Update job data
	updatedJobData, _ := json.Marshal(job)
	pipe.HSet(ctx, jobKey, map[string]interface{}{
		hashFieldJob:    string(updatedJobData),
		hashFieldStatus: string(job.Status),
	})

	// Move from processing to completed queue
	pipe.ZRem(ctx, processingKey, jobID)
	pipe.ZAdd(ctx, completedKey, redis.Z{
		Score:  float64(now.UnixNano()),
		Member: jobID,
	})

	// Execute pipeline
	if _, err := pipe.Exec(ctx); err != nil {
		return fmt.Errorf("failed to complete job: %w", err)
	}

	// Record metrics
	metrics.JobsProcessed.WithLabelValues(string(job.Type), string(domain.JobStatusCompleted)).Inc()
	metrics.JobStatusTransitions.WithLabelValues(string(job.Type), string(prevStatus), string(domain.JobStatusCompleted)).Inc()
	if job.StartedAt != nil {
		metrics.JobProcessingDuration.WithLabelValues(string(job.Type)).Observe(time.Since(*job.StartedAt).Seconds())
	}

	return nil
}

// Fail marks a job as failed
func (q *RedisQueue) Fail(ctx context.Context, jobID string, jobErr error) error {
	jobKey := q.jobKey(jobID)
	processingKey := q.queueKey(queueKeyProcessing)
	failedKey := q.queueKey(queueKeyFailed)

	// Get current job data
	jobData, err := q.client.HGet(ctx, jobKey, hashFieldJob).Result()
	if err != nil {
		return fmt.Errorf("failed to get job data: %w", err)
	}

	// Deserialize job
	var job domain.Job
	if err := json.Unmarshal([]byte(jobData), &job); err != nil {
		return fmt.Errorf("failed to unmarshal job: %w", err)
	}

	// Update job status
	now := time.Now()
	prevStatus := job.Status
	job.Status = domain.JobStatusFailed
	job.Error = jobErr.Error()
	job.RetryCount++

	// Check if should retry
	var nextRetryAt time.Time
	if job.RetryCount < job.MaxRetries {
		// Calculate next retry time with exponential backoff
		delay := q.config.RetryDelay * time.Duration(1<<uint(job.RetryCount-1))
		if delay > q.config.MaxRetryDelay {
			delay = q.config.MaxRetryDelay
		}
		nextRetryAt = now.Add(delay)
		job.NextRetryAt = &nextRetryAt
		job.Status = domain.JobStatusPending
	}

	// Start pipeline
	pipe := q.client.Pipeline()

	// Update job data
	updatedJobData, _ := json.Marshal(job)
	pipe.HSet(ctx, jobKey, map[string]interface{}{
		hashFieldJob:        string(updatedJobData),
		hashFieldStatus:     string(job.Status),
		hashFieldError:      job.Error,
		hashFieldRetryCount: job.RetryCount,
	})

	// Move from processing queue
	pipe.ZRem(ctx, processingKey, jobID)

	// Add to appropriate queue based on retry status
	if job.Status == domain.JobStatusPending {
		// Add back to pending queue with retry time as score
		pipe.ZAdd(ctx, q.queueKey(queueKeyPending), redis.Z{
			Score:  float64(nextRetryAt.UnixNano()),
			Member: jobID,
		})
	} else {
		// Add to failed queue
		pipe.ZAdd(ctx, failedKey, redis.Z{
			Score:  float64(now.UnixNano()),
			Member: jobID,
		})
	}

	// Execute pipeline
	if _, err := pipe.Exec(ctx); err != nil {
		return fmt.Errorf("failed to fail job: %w", err)
	}

	// Record metrics
	metrics.JobsProcessed.WithLabelValues(string(job.Type), string(domain.JobStatusFailed)).Inc()
	metrics.JobStatusTransitions.WithLabelValues(string(job.Type), string(prevStatus), string(job.Status)).Inc()
	metrics.JobRetries.WithLabelValues(string(job.Type)).Add(float64(job.RetryCount))
	metrics.JobErrors.WithLabelValues(string(job.Type), job.Error).Inc()

	return nil
}

// Cancel cancels a pending or running job
func (q *RedisQueue) Cancel(ctx context.Context, jobID string) error {
	jobKey := q.jobKey(jobID)

	// Get current job data
	jobData, err := q.client.HGet(ctx, jobKey, hashFieldJob).Result()
	if err != nil {
		return fmt.Errorf("failed to get job data: %w", err)
	}

	// Deserialize job
	var job domain.Job
	if err := json.Unmarshal([]byte(jobData), &job); err != nil {
		return fmt.Errorf("failed to unmarshal job: %w", err)
	}

	// Can only cancel pending or processing jobs
	if job.Status != domain.JobStatusPending && job.Status != domain.JobStatusProcessing {
		return fmt.Errorf("cannot cancel job with status %s", job.Status)
	}

	prevStatus := job.Status
	job.Status = domain.JobStatusCanceled

	// Start pipeline
	pipe := q.client.Pipeline()

	// Update job data
	updatedJobData, _ := json.Marshal(job)
	pipe.HSet(ctx, jobKey, map[string]interface{}{
		hashFieldJob:    string(updatedJobData),
		hashFieldStatus: string(job.Status),
	})

	// Remove from current queue
	if prevStatus == domain.JobStatusPending {
		pipe.ZRem(ctx, q.queueKey(queueKeyPending), jobID)
	} else {
		pipe.ZRem(ctx, q.queueKey(queueKeyProcessing), jobID)
	}

	// Execute pipeline
	if _, err := pipe.Exec(ctx); err != nil {
		return fmt.Errorf("failed to cancel job: %w", err)
	}

	// Record metrics
	metrics.JobStatusTransitions.WithLabelValues(string(job.Type), string(prevStatus), string(domain.JobStatusCanceled)).Inc()
	if prevStatus == domain.JobStatusPending {
		metrics.JobsInQueue.WithLabelValues(string(job.Type)).Dec()
	}

	return nil
}

// GetStatus gets the current status of a job
func (q *RedisQueue) GetStatus(ctx context.Context, jobID string) (*domain.Job, error) {
	jobKey := q.jobKey(jobID)

	// Get job data
	jobData, err := q.client.HGet(ctx, jobKey, hashFieldJob).Result()
	if err != nil {
		if err == redis.Nil {
			return nil, fmt.Errorf("job not found: %s", jobID)
		}
		return nil, fmt.Errorf("failed to get job data: %w", err)
	}

	// Deserialize job
	var job domain.Job
	if err := json.Unmarshal([]byte(jobData), &job); err != nil {
		return nil, fmt.Errorf("failed to unmarshal job: %w", err)
	}

	return &job, nil
}

// UpdateProgress updates the progress of a running job
func (q *RedisQueue) UpdateProgress(ctx context.Context, jobID string, progress int) error {
	jobKey := q.jobKey(jobID)

	// Get current job data
	jobData, err := q.client.HGet(ctx, jobKey, hashFieldJob).Result()
	if err != nil {
		return fmt.Errorf("failed to get job data: %w", err)
	}

	// Deserialize job
	var job domain.Job
	if err := json.Unmarshal([]byte(jobData), &job); err != nil {
		return fmt.Errorf("failed to unmarshal job: %w", err)
	}

	// Can only update progress of processing jobs
	if job.Status != domain.JobStatusProcessing {
		return fmt.Errorf("cannot update progress of job with status %s", job.Status)
	}

	// Update progress
	job.Progress = progress

	// Update job data
	updatedJobData, _ := json.Marshal(job)
	if err := q.client.HSet(ctx, jobKey, hashFieldJob, string(updatedJobData), hashFieldProgress, progress).Err(); err != nil {
		return fmt.Errorf("failed to update job progress: %w", err)
	}

	return nil
}
