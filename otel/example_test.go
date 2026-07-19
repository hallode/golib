package otel_test

import (
	"context"
	"fmt"

	"github.com/hallode/golib/v2/otel"
)

func ExampleNewTracer() {
	_, err := otel.NewTracer(&otel.TracerConfig{
		Name:         "orders",
		Tracer:       otel.ExporterOTLP,
		OtelEndpoint: "http://localhost:4318", // OTLP HTTP, not gRPC :4317
	})
	if err != nil {
		fmt.Println(err)
		return
	}

	// Call Tracer directly at the top of an instrumented function — it derives
	// the span name from its caller, so wrapping it breaks span naming.
	_, span := otel.Tracer(context.Background())
	defer span.End()
}
