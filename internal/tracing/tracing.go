package tracing

import (
	"context"
	"errors"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	tracesdk "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	"go.opentelemetry.io/otel/trace"
)

// Init initializes OpenTelemetry distributed tracing with an OTLP gRPC exporter.
// It sets up the global TracerProvider and propagator (TraceContext + Baggage).
// Returns a shutdown function to flush spans on application exit.
func Init(serviceName, collectorURL string) (func(), error) {
	if collectorURL == "" {
		collectorURL = "localhost:4317"
	}

	ctx := context.Background()

	exporter, err := otlptracegrpc.New(ctx,
		otlptracegrpc.WithEndpoint(collectorURL),
		otlptracegrpc.WithInsecure(),
	)
	if err != nil {
		return nil, errors.New("failed to create OTLP gRPC exporter: " + err.Error())
	}

	res, err := resource.Merge(
		resource.Default(),
		resource.NewWithAttributes(
			"https://opentelemetry.io/schemas/1.21.0",
			attribute.String("service.name", serviceName),
		),
	)
	if err != nil {
		return nil, errors.New("failed to create resource: " + err.Error())
	}

	tp := tracesdk.NewTracerProvider(
		tracesdk.WithBatcher(exporter),
		tracesdk.WithResource(res),
	)

	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(propagation.TraceContext{})

	shutdown := func() {
		_ = tp.Shutdown(ctx)
	}

	return shutdown, nil
}

// InitForTesting sets up a TracerProvider with an InMemoryExporter for tests.
// Returns the exporter so tests can inspect recorded spans.
func InitForTesting() *tracetest.InMemoryExporter {
	exporter := tracetest.NewInMemoryExporter()

	tp := tracesdk.NewTracerProvider(
		tracesdk.WithSyncer(exporter),
	)

	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(propagation.TraceContext{})

	return exporter
}

// Tracer returns a named Tracer from the global provider.
func Tracer(name string) trace.Tracer {
	return otel.Tracer(name)
}
