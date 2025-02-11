package tracing

import (
	"context"
	"fmt"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.4.0"
)

// Config holds configuration for OpenTelemetry tracing
type Config struct {
	Enabled     bool          `json:"enabled"`
	ServiceName string        `json:"service_name"`
	Endpoint    string        `json:"endpoint"`
	SampleRate  float64       `json:"sample_rate"`
	BatchSize   int           `json:"batch_size"`
	Timeout     time.Duration `json:"timeout"`
}

// Provider manages the OpenTelemetry tracer provider
type Provider struct {
	tp *sdktrace.TracerProvider
}

// NewProvider creates a new OpenTelemetry tracer provider
func NewProvider(ctx context.Context, cfg *Config) (*Provider, error) {
	if !cfg.Enabled {
		return &Provider{}, nil
	}

	exporter, err := otlptracegrpc.New(ctx,
		otlptracegrpc.WithEndpoint(cfg.Endpoint),
		otlptracegrpc.WithInsecure(),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create exporter: %w", err)
	}

	res, err := resource.New(ctx,
		resource.WithAttributes(
			semconv.ServiceNameKey.String(cfg.ServiceName),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create resource: %w", err)
	}

	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(res),
		sdktrace.WithSampler(sdktrace.TraceIDRatioBased(cfg.SampleRate)),
	)
	otel.SetTracerProvider(tp)

	return &Provider{tp: tp}, nil
}

// Shutdown cleanly shuts down the tracer provider
func (p *Provider) Shutdown(ctx context.Context) error {
	if p.tp == nil {
		return nil
	}
	return p.tp.Shutdown(ctx)
}

// Tracer returns a named tracer
func (p *Provider) Tracer(name string) *sdktrace.TracerProvider {
	return p.tp
}
