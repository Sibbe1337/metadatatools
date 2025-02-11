package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// QueueMetrics holds metrics for queue operations
type QueueMetrics struct {
	// Publishing metrics
	MessagesPublished *prometheus.CounterVec
	PublishErrors     *prometheus.CounterVec
	PublishLatency    *prometheus.HistogramVec

	// Processing metrics
	MessagesProcessed  *prometheus.CounterVec
	ProcessingErrors   *prometheus.CounterVec
	ProcessingLatency  *prometheus.HistogramVec
	DeadLetters        *prometheus.CounterVec
	SubscriptionErrors *prometheus.CounterVec
}

// NewQueueMetrics creates a new QueueMetrics instance
func NewQueueMetrics() *QueueMetrics {
	return &QueueMetrics{
		MessagesPublished: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "queue_messages_published_total",
				Help: "Total number of messages published",
			},
			[]string{"topic"},
		),
		PublishErrors: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "queue_publish_errors_total",
				Help: "Total number of publish errors",
			},
			[]string{"topic"},
		),
		PublishLatency: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "queue_publish_latency_seconds",
				Help:    "Time taken to publish messages",
				Buckets: []float64{0.01, 0.05, 0.1, 0.5, 1, 2, 5},
			},
			[]string{"topic"},
		),
		MessagesProcessed: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "queue_messages_processed_total",
				Help: "Total number of messages processed",
			},
			[]string{"subscription"},
		),
		ProcessingErrors: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "queue_processing_errors_total",
				Help: "Total number of processing errors",
			},
			[]string{"subscription"},
		),
		ProcessingLatency: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "queue_processing_latency_seconds",
				Help:    "Time taken to process messages",
				Buckets: []float64{0.1, 0.5, 1, 2, 5, 10, 30},
			},
			[]string{"subscription"},
		),
		DeadLetters: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "queue_dead_letters_total",
				Help: "Total number of messages sent to dead letter queue",
			},
			[]string{"type"},
		),
		SubscriptionErrors: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "queue_subscription_errors_total",
				Help: "Total number of subscription errors",
			},
			[]string{"subscription"},
		),
	}
}

var (
	// Queue operation metrics
	QueueOperations = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "queue_operations_total",
		Help: "Total number of queue operations",
	}, []string{"operation", "topic", "status"}) // operation: publish, subscribe, ack, nack; status: success, failure

	// Message processing metrics
	MessageProcessingDuration = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "message_processing_duration_seconds",
		Help:    "Duration of message processing",
		Buckets: []float64{.005, .01, .025, .05, .1, .25, .5, 1, 2.5, 5, 10},
	}, []string{"topic"})

	// Message status metrics
	MessageStatus = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "messages_by_status",
		Help: "Number of messages by status",
	}, []string{"topic", "status"}) // status: pending, processing, completed, failed, retrying, dead_letter

	// Retry metrics
	MessageRetries = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "message_retries_total",
		Help: "Total number of message retry attempts",
	}, []string{"topic"})

	// Dead letter metrics
	DeadLetterMessages = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "dead_letter_messages",
		Help: "Number of messages in dead letter queue",
	}, []string{"topic"})

	// Queue size metrics
	QueueSize = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "queue_size",
		Help: "Current size of the queue",
	}, []string{"topic"})

	// Processing error metrics
	ProcessingErrors = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "processing_errors_total",
		Help: "Total number of processing errors",
	}, []string{"topic", "error_type"}) // error_type: timeout, validation, processing, etc.

	// Batch processing metrics
	BatchProcessingDuration = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "batch_processing_duration_seconds",
		Help:    "Duration of batch message processing",
		Buckets: []float64{.01, .025, .05, .1, .25, .5, 1, 2.5, 5, 10},
	}, []string{"topic"})
)
