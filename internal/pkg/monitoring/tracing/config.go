package tracing

import (
	"fmt"
	"time"
)

// Config holds configuration for OpenTelemetry tracing
type Config struct {
	ServiceName    string        `env:"OTEL_SERVICE_NAME,required"`
	CollectorURL   string        `env:"OTEL_COLLECTOR_URL" envDefault:"http://localhost:4318"`
	SampleRate     float64       `env:"OTEL_SAMPLE_RATE" envDefault:"1.0"`
	Timeout        time.Duration `env:"OTEL_TIMEOUT" envDefault:"5s"`
	BatchSize      int           `env:"OTEL_BATCH_SIZE" envDefault:"100"`
	ExportInterval time.Duration `env:"OTEL_EXPORT_INTERVAL" envDefault:"1s"`
}

// Validate checks if the configuration is valid
func (c *Config) Validate() error {
	if c.ServiceName == "" {
		return fmt.Errorf("service name is required")
	}
	if c.SampleRate < 0 || c.SampleRate > 1 {
		return fmt.Errorf("sample rate must be between 0 and 1")
	}
	return nil
}
