package postgres

import (
	"context"

	"github.com/jackc/pgx/v5"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

// otelQueryTracer implements pgx.QueryTracer, emitting one span per query.
// When tracing isn't configured (see internal/infrastructure/tracing),
// otel.Tracer returns a no-op tracer, so this has near-zero overhead.
type otelQueryTracer struct {
	tracer trace.Tracer
}

// newOtelQueryTracer creates a pgx.QueryTracer backed by the global otel
// TracerProvider.
func newOtelQueryTracer() pgx.QueryTracer {
	return &otelQueryTracer{tracer: otel.Tracer("goledger/pgx")}
}

type spanCtxKey struct{}

func (t *otelQueryTracer) TraceQueryStart(ctx context.Context, _ *pgx.Conn, data pgx.TraceQueryStartData) context.Context {
	ctx, span := t.tracer.Start(ctx, "pgx.query",
		trace.WithSpanKind(trace.SpanKindClient),
		trace.WithAttributes(attribute.String("db.system", "postgresql"), attribute.String("db.statement", data.SQL)),
	)

	return context.WithValue(ctx, spanCtxKey{}, span)
}

func (t *otelQueryTracer) TraceQueryEnd(ctx context.Context, _ *pgx.Conn, data pgx.TraceQueryEndData) {
	span, ok := ctx.Value(spanCtxKey{}).(trace.Span)
	if !ok {
		return
	}
	defer span.End()

	if data.Err != nil {
		span.RecordError(data.Err)
		span.SetStatus(codes.Error, data.Err.Error())

		return
	}

	span.SetAttributes(attribute.String("db.rows_affected", data.CommandTag.String()))
}

var _ pgx.QueryTracer = (*otelQueryTracer)(nil)
