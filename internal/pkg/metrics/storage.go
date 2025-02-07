package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	// StorageOperationDuration tracks the duration of storage operations
	StorageOperationDuration = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "storage_operation_duration_seconds",
		Help:    "Duration of storage operations",
		Buckets: []float64{0.1, 0.5, 1, 2, 5, 10},
	}, []string{"operation"})

	// StorageOperationErrors tracks the number of storage operation errors
	StorageOperationErrors = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "storage_operation_errors_total",
		Help: "Total number of storage operation errors",
	}, []string{"operation"})

	// StorageOperationSuccess tracks the number of successful storage operations
	StorageOperationSuccess = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "storage_operation_success_total",
		Help: "Total number of successful storage operations",
	}, []string{"operation"})

	// StorageQuotaUsage tracks the storage quota usage per user
	StorageQuotaUsage = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "storage_quota_usage_bytes",
		Help: "Current storage quota usage in bytes",
	}, []string{"user_id"})

	// StorageTempFileCount tracks the number of temporary files
	StorageTempFileCount = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "storage_temp_file_count",
		Help: "Number of temporary files in storage",
	})
)
