package telemetry

import (
	"context"
	"fmt"
	"os"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/jaeger"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.4.0"
	"go.opentelemetry.io/otel/trace"
)

// Config holds the telemetry configuration
type Config struct {
	ServiceName    string
	ServiceVersion string
	Environment    string
	EnableTracing  bool
	JaegerEndpoint string
	TempoEndpoint  string
	SampleRate     float64
}

// InitTracer initializes the OpenTelemetry tracer
func InitTracer(cfg Config) (*sdktrace.TracerProvider, error) {
	if !cfg.EnableTracing {
		return nil, nil
	}

	// Ensure default values
	if cfg.ServiceName == "" {
		cfg.ServiceName = "cosmo-backend"
	}
	if cfg.SampleRate == 0 {
		cfg.SampleRate = 0.1 // 10% sample rate by default
	}

	// Create resource with service information
	res, err := resource.New(context.Background(),
		resource.WithAttributes(
			semconv.ServiceNameKey.String(cfg.ServiceName),
			semconv.ServiceVersionKey.String(cfg.ServiceVersion),
			semconv.DeploymentEnvironmentKey.String(cfg.Environment),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create resource: %w", err)
	}

	// Create trace exporter
	var exporter sdktrace.SpanExporter

	// Try Tempo first, then fallback to Jaeger
	if cfg.TempoEndpoint != "" {
		exporter, err = otlptracehttp.New(context.Background(),
			otlptracehttp.WithEndpoint(cfg.TempoEndpoint),
			otlptracehttp.WithInsecure(),
		)
	} else if cfg.JaegerEndpoint != "" {
		exporter, err = jaeger.New(jaeger.WithCollectorEndpoint(jaeger.WithEndpoint(cfg.JaegerEndpoint)))
	} else {
		// Default to localhost Jaeger
		exporter, err = jaeger.New(jaeger.WithCollectorEndpoint(jaeger.WithEndpoint("http://localhost:14268/api/traces")))
	}

	if err != nil {
		return nil, fmt.Errorf("failed to create trace exporter: %w", err)
	}

	// Create tracer provider
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(res),
		sdktrace.WithSampler(sdktrace.TraceIDRatioBased(cfg.SampleRate)),
	)

	// Register as global tracer provider
	otel.SetTracerProvider(tp)

	// Set global propagator
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))

	return tp, nil
}

// StartSpan starts a new span with the given name and options
func StartSpan(ctx context.Context, name string, opts ...trace.SpanStartOption) (context.Context, trace.Span) {
	tracer := otel.Tracer(os.Getenv("OTEL_SERVICE_NAME"))
	return tracer.Start(ctx, name, opts...)
}

// AddSpanAttributes adds attributes to the current span
func AddSpanAttributes(ctx context.Context, attrs ...attribute.KeyValue) {
	span := trace.SpanFromContext(ctx)
	if span != nil {
		span.SetAttributes(attrs...)
	}
}

// AddSpanEvent adds an event to the current span
func AddSpanEvent(ctx context.Context, name string, attrs ...attribute.KeyValue) {
	span := trace.SpanFromContext(ctx)
	if span != nil {
		span.AddEvent(name, trace.WithAttributes(attrs...))
	}
}

// RecordError records an error on the current span
func RecordError(ctx context.Context, err error) {
	if err == nil {
		return
	}

	span := trace.SpanFromContext(ctx)
	if span != nil {
		span.RecordError(err)
		span.SetAttributes(attribute.String("error", err.Error()))
	}
}

// SpanFromContext returns the current span from context
func SpanFromContext(ctx context.Context) trace.Span {
	return trace.SpanFromContext(ctx)
}

// WithSpan creates a new span and runs the given function
func WithSpan(ctx context.Context, name string, fn func(context.Context) error) error {
	ctx, span := StartSpan(ctx, name)
	defer span.End()

	err := fn(ctx)
	if err != nil {
		RecordError(ctx, err)
		span.SetAttributes(attribute.String("error", err.Error()))
	}

	return err
}

// Shutdown gracefully shuts down the tracer provider
func Shutdown(tp *sdktrace.TracerProvider, timeout time.Duration) error {
	if tp == nil {
		return nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	return tp.Shutdown(ctx)
}
