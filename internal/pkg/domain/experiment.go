package domain

// ExperimentMetrics holds metrics for both control and experiment groups
type ExperimentMetrics struct {
	Control    *ModelMetrics
	Experiment *ModelMetrics
}

// ModelMetrics holds metrics for a single model
type ModelMetrics struct {
	TotalRequests     int64
	AvgProcessingTime float64
	AvgConfidence     float64
	SuccessRate       float64
}
