package internal

import (
	"context"
	"log/slog"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"
)

func SetupTracing() *sdktrace.TracerProvider {
	provider := sdktrace.NewTracerProvider(
		sdktrace.WithSampler(
			sdktrace.ParentBased(sdktrace.AlwaysSample()),
		),
	)

	otel.SetTracerProvider(provider)
	otel.SetTextMapPropagator(
		propagation.NewCompositeTextMapPropagator(
			propagation.TraceContext{},
			propagation.Baggage{},
		),
	)

	return provider
}

type TraceLogHandler struct {
	slog.Handler
	traceIDBound bool
}

func hasAttr(record slog.Record, key string) bool {
	found := false

	record.Attrs(func(attr slog.Attr) bool {
		if attr.Key == key {
			found = true
			return false
		}
		return true
	})

	return found
}

func (h TraceLogHandler) Handle(ctx context.Context, record slog.Record) error {
	sc := trace.SpanContextFromContext(ctx)

	if sc.IsValid() && !h.traceIDBound {
		if !hasAttr(record, "trace_id") {
			record.AddAttrs(
				slog.String("trace_id", sc.TraceID().String()),
			)
		}
	}

	return h.Handler.Handle(ctx, record)
}

func (h TraceLogHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	traceIDBound := h.traceIDBound
	for _, attr := range attrs {
		if attr.Key == "trace_id" {
			traceIDBound = true
			break
		}
	}

	return TraceLogHandler{
		Handler:      h.Handler.WithAttrs(attrs),
		traceIDBound: traceIDBound,
	}
}

func (h TraceLogHandler) WithGroup(name string) slog.Handler {
	return TraceLogHandler{
		Handler:      h.Handler.WithGroup(name),
		traceIDBound: h.traceIDBound,
	}
}
