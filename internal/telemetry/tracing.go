// Package telemetry provides OpenTelemetry tracing functionality
package telemetry

import (
	"context"
	"fmt"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.21.0"
	"go.opentelemetry.io/otel/trace"
)

// TracingConfig contains OpenTelemetry tracing configuration
type TracingConfig struct {
	Enabled        bool
	Endpoint       string // OTLP endpoint (e.g., "tempo:4318")
	ServiceName    string
	ServiceVersion string
	Environment    string
	SamplingRatio  float64 // 0.0 to 1.0, where 1.0 = 100% sampling
	InsecureConn   bool    // Use insecure connection
}

// TracerProvider wraps the OpenTelemetry tracer provider
type TracerProvider struct {
	provider *sdktrace.TracerProvider
	tracer   trace.Tracer
}

// InitTracing initializes OpenTelemetry tracing
func InitTracing(ctx context.Context, config TracingConfig) (*TracerProvider, error) {
	if !config.Enabled {
		// Return a no-op tracer provider
		return &TracerProvider{
			provider: sdktrace.NewTracerProvider(),
			tracer:   otel.Tracer(config.ServiceName),
		}, nil
	}

	// Create OTLP HTTP exporter
	opts := []otlptracehttp.Option{
		otlptracehttp.WithEndpoint(config.Endpoint),
	}

	if config.InsecureConn {
		opts = append(opts, otlptracehttp.WithInsecure())
	}

	exporter, err := otlptracehttp.New(ctx, opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to create OTLP exporter: %w", err)
	}

	// Create resource with service information
	res, err := resource.New(ctx,
		resource.WithAttributes(
			semconv.ServiceName(config.ServiceName),
			semconv.ServiceVersion(config.ServiceVersion),
			semconv.DeploymentEnvironment(config.Environment),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create resource: %w", err)
	}

	// Create sampler based on sampling ratio
	var sampler sdktrace.Sampler
	if config.SamplingRatio >= 1.0 {
		sampler = sdktrace.AlwaysSample()
	} else if config.SamplingRatio <= 0.0 {
		sampler = sdktrace.NeverSample()
	} else {
		sampler = sdktrace.TraceIDRatioBased(config.SamplingRatio)
	}

	// Create tracer provider
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter,
			sdktrace.WithMaxExportBatchSize(512),
			sdktrace.WithBatchTimeout(5*time.Second),
			sdktrace.WithMaxQueueSize(2048),
		),
		sdktrace.WithResource(res),
		sdktrace.WithSampler(sampler),
	)

	// Set global tracer provider
	otel.SetTracerProvider(tp)

	// Set global propagator for context propagation (W3C Trace Context)
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))

	// Create tracer
	tracer := otel.Tracer(config.ServiceName)

	return &TracerProvider{
		provider: tp,
		tracer:   tracer,
	}, nil
}

// Shutdown gracefully shuts down the tracer provider
func (tp *TracerProvider) Shutdown(ctx context.Context) error {
	if tp.provider == nil {
		return nil
	}
	return tp.provider.Shutdown(ctx)
}

// Tracer returns the configured tracer
func (tp *TracerProvider) Tracer() trace.Tracer {
	return tp.tracer
}

// GetTracer returns the global tracer
func GetTracer(name string) trace.Tracer {
	return otel.Tracer(name)
}
