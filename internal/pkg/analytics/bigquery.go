// Package analytics provides analytics and metrics collection functionality.
//
// This package implements analytics services for tracking and analyzing AI
// experiment results, using Google BigQuery as the backend storage.
//
// Key features:
//   - Recording AI experiment results
//   - Calculating experiment metrics
//   - Comparing control and experiment groups
//
// Usage example:
//
//	service, err := analytics.NewBigQueryService(projectID, dataset)
//	if err != nil {
//	    log.Fatal(err)
//	}
//	defer service.Close()
//
//	record := &analytics.AIExperimentRecord{
//	    Timestamp:       time.Now(),
//	    TrackID:        "track123",
//	    ModelProvider:  "openai",
//	    ModelVersion:   "v1",
//	    ProcessingTime: 1.5,
//	    Confidence:    0.95,
//	    Success:       true,
//	}
//
//	if err := service.RecordAIExperiment(ctx, record); err != nil {
//	    log.Printf("Failed to record experiment: %v", err)
//	}
package analytics

import (
	"context"
	"fmt"
	"metadatatool/internal/pkg/domain"
	"time"

	"cloud.google.com/go/bigquery"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
)

// AIExperimentRecord represents a record of an AI experiment.
// Each record contains information about a single AI operation,
// including timing, confidence scores, and success/failure status.
type AIExperimentRecord struct {
	// Timestamp when the experiment was conducted
	Timestamp time.Time

	// TrackID is the unique identifier of the processed track
	TrackID string

	// ModelProvider identifies the AI model provider (e.g., "openai", "qwen2")
	ModelProvider string

	// ModelVersion identifies the specific version of the model used
	ModelVersion string

	// ProcessingTime is the time taken to process the track in seconds
	ProcessingTime float64

	// Confidence is the model's confidence score for the operation (0-1)
	Confidence float64

	// Success indicates whether the operation was successful
	Success bool

	// ErrorMessage contains error details if Success is false
	ErrorMessage string

	// ExperimentGroup indicates whether this was part of the control or experiment group
	ExperimentGroup string // "control" or "experiment"
}

// BigQueryService handles analytics data storage in Google BigQuery.
// It provides methods for recording experiment results and retrieving
// aggregated metrics for analysis.
type BigQueryService struct {
	client          *bigquery.Client
	projectID       string
	dataset         string
	experimentTable *bigquery.Table
}

// NewBigQueryService creates a new BigQuery service instance.
// It establishes a connection to BigQuery and initializes the required
// dataset and table references.
//
// Parameters:
//   - projectID: Google Cloud project ID
//   - dataset: BigQuery dataset name
//
// Returns:
//   - *BigQueryService: Initialized service
//   - error: Any error that occurred during initialization
func NewBigQueryService(projectID, dataset string) (*BigQueryService, error) {
	ctx := context.Background()
	client, err := bigquery.NewClient(ctx, projectID, option.WithScopes(bigquery.Scope))
	if err != nil {
		return nil, fmt.Errorf("failed to create BigQuery client: %w", err)
	}

	ds := client.Dataset(dataset)
	table := ds.Table("ai_experiments")

	return &BigQueryService{
		client:          client,
		projectID:       projectID,
		dataset:         dataset,
		experimentTable: table,
	}, nil
}

// RecordAIExperiment records an AI experiment result in BigQuery.
// This method is used to track individual AI operations for later analysis.
//
// Parameters:
//   - ctx: Context for the operation
//   - record: The experiment record to store
//
// Returns:
//   - error: Any error that occurred during the operation
func (s *BigQueryService) RecordAIExperiment(ctx context.Context, record *AIExperimentRecord) error {
	inserter := s.experimentTable.Inserter()
	return inserter.Put(ctx, record)
}

// GetExperimentMetrics retrieves aggregated metrics for the experiment.
// This method calculates various metrics for both control and experiment groups
// within the specified time range.
//
// Parameters:
//   - ctx: Context for the operation
//   - start: Start time for the analysis period
//   - end: End time for the analysis period
//
// Returns:
//   - *domain.ExperimentMetrics: Aggregated metrics for both groups
//   - error: Any error that occurred during the operation
func (s *BigQueryService) GetExperimentMetrics(ctx context.Context, start, end time.Time) (*domain.ExperimentMetrics, error) {
	query := fmt.Sprintf(`
		SELECT
			ExperimentGroup,
			COUNT(*) as TotalRequests,
			AVG(ProcessingTime) as AvgProcessingTime,
			AVG(Confidence) as AvgConfidence,
			COUNTIF(Success) / COUNT(*) as SuccessRate
		FROM %s.ai_experiments
		WHERE Timestamp BETWEEN @start AND @end
		GROUP BY ExperimentGroup
	`, s.dataset)

	q := s.client.Query(query)
	q.Parameters = []bigquery.QueryParameter{
		{Name: "start", Value: start},
		{Name: "end", Value: end},
	}

	it, err := q.Read(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to execute query: %w", err)
	}

	metrics := &domain.ExperimentMetrics{
		Control:    &domain.ModelMetrics{},
		Experiment: &domain.ModelMetrics{},
	}

	for {
		var row struct {
			ExperimentGroup   string
			TotalRequests     int64
			AvgProcessingTime float64
			AvgConfidence     float64
			SuccessRate       float64
		}
		err := it.Next(&row)
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("failed to read row: %w", err)
		}

		if row.ExperimentGroup == "control" {
			metrics.Control.TotalRequests = row.TotalRequests
			metrics.Control.AvgProcessingTime = row.AvgProcessingTime
			metrics.Control.AvgConfidence = row.AvgConfidence
			metrics.Control.SuccessRate = row.SuccessRate
		} else {
			metrics.Experiment.TotalRequests = row.TotalRequests
			metrics.Experiment.AvgProcessingTime = row.AvgProcessingTime
			metrics.Experiment.AvgConfidence = row.AvgConfidence
			metrics.Experiment.SuccessRate = row.SuccessRate
		}
	}

	return metrics, nil
}

// Close closes the BigQuery client and releases associated resources.
// This method should be called when the service is no longer needed.
func (s *BigQueryService) Close() error {
	return s.client.Close()
}
