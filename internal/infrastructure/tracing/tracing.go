// Package tracing wires up OpenTelemetry distributed tracing across the
// HTTP and gRPC APIs and the pgx database driver, correlated with the same
// request_id used in structured logs and audit rows (see domain.RequestMeta)
// so a single transfer can be followed request -> spans -> audit -> outbox.
package tracing

import (
	"context"
	"fmt"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/exporters/stdout/stdouttrace"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
)

// Config controls tracer provider setup.
type Config struct {
	// Enabled turns tracing on. When false, Setup is a no-op and the
	// process-wide otel tracer stays the default no-op implementation, so
	// Tracer(...).Start(...) calls elsewhere are effectively free.
	Enabled bool
	// ServiceName identifies this process in trace backends.
	ServiceName string
	// OTLPEndpoint is the OTLP/gRPC collector address (host:port, no
	// scheme), e.g. "localhost:4317". Empty uses a stdout exporter instead
	// - useful for local development without a collector running.
	OTLPEndpoint string
}

// Setup configures the global TracerProvider and propagator per cfg. The
// returned shutdown func flushes and closes the exporter; call it during
// graceful shutdown. Returns a no-op shutdown when tracing is disabled.
func Setup(ctx context.Context, cfg Config) (shutdown func(context.Context) error, err error) {
	noop := func(context.Context) error { return nil }

	if !cfg.Enabled {
		return noop, nil
	}

	res, err := resource.Merge(
		resource.Default(),
		resource.NewSchemaless(semconv.ServiceName(cfg.ServiceName)),
	)
	if err != nil {
		return noop, fmt.Errorf("failed to build tracing resource: %w", err)
	}

	exporter, err := newExporter(ctx, cfg)
	if err != nil {
		return noop, fmt.Errorf("failed to create trace exporter: %w", err)
	}

	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(res),
	)

	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))

	return tp.Shutdown, nil
}

func newExporter(ctx context.Context, cfg Config) (sdktrace.SpanExporter, error) {
	if cfg.OTLPEndpoint == "" {
		return stdouttrace.New(stdouttrace.WithPrettyPrint())
	}

	return otlptracegrpc.New(ctx,
		otlptracegrpc.WithEndpoint(cfg.OTLPEndpoint),
		otlptracegrpc.WithInsecure(),
	)
}
