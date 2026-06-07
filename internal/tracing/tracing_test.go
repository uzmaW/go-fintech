package tracing

import (
	"context"
	"testing"
)

func TestInitForTesting(t *testing.T) {
	exporter := InitForTesting()
	if exporter == nil {
		t.Fatal("InitForTesting returned nil exporter")
	}
}

func TestTracerCreation(t *testing.T) {
	InitForTesting()
	tracer := Tracer("test-service")
	if tracer == nil {
		t.Fatal("Tracer returned nil")
	}
}

func TestSpanCreation(t *testing.T) {
	exporter := InitForTesting()
	tracer := Tracer("test-service")

	_, span := tracer.Start(context.Background(), "test-span")
	span.End()

	spans := exporter.GetSpans()
	if len(spans) != 1 {
		t.Fatalf("expected 1 span, got %d", len(spans))
	}
	if spans[0].Name != "test-span" {
		t.Errorf("expected span name 'test-span', got %q", spans[0].Name)
	}
}
