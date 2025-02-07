package analytics

import (
	"context"
	"fmt"
	"time"

	"metadatatool/internal/pkg/domain"

	"cloud.google.com/go/bigquery"
	"google.golang.org/api/iterator"
)

// AIExperimentRecord represents a single AI processing record
type AIExperimentRecord struct {
	Timestamp       time.Time
	TrackID         string
	ModelProvider   string
	ModelVersion    string
	ProcessingTime  float64
	Confidence      float64
	Success         bool
	ErrorMessage    string
	ExperimentGroup string // "control" or "experiment"
}

// BigQueryService handles analytics data storage
type BigQueryService struct {
	client          *bigquery.Client
	projectID       string
	dataset         string
	experimentTable *bigquery.Table
}

// NewBigQueryService creates a new BigQuery service
func NewBigQueryService(projectID, dataset string) (*BigQueryService, error) {
	ctx := context.Background()
	client, err := bigquery.NewClient(ctx, projectID)
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

// RecordAIExperiment records an AI processing experiment
func (s *BigQueryService) RecordAIExperiment(ctx context.Context, record *AIExperimentRecord) error {
	inserter := s.experimentTable.Inserter()
	return inserter.Put(ctx, record)
}

// GetExperimentMetrics retrieves experiment metrics for a time range
func (s *BigQueryService) GetExperimentMetrics(ctx context.Context, start, end time.Time) (*domain.ExperimentMetrics, error) {
	query := s.client.Query(`
		SELECT
			experiment_group,
			COUNT(*) as total_requests,
			AVG(processing_time) as avg_processing_time,
			AVG(confidence) as avg_confidence,
			COUNTIF(success) / COUNT(*) as success_rate
		FROM ` + "`" + s.projectID + "." + s.dataset + ".ai_experiments" + "`" + `
		WHERE timestamp BETWEEN @start AND @end
		GROUP BY experiment_group
	`)

	query.Parameters = []bigquery.QueryParameter{
		{Name: "start", Value: start},
		{Name: "end", Value: end},
	}

	it, err := query.Read(ctx)
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

// Close closes the BigQuery client
func (s *BigQueryService) Close() error {
	return s.client.Close()
}
