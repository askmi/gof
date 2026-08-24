package internal

import (
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
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
