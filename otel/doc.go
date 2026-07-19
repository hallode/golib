// Package otel sets up OpenTelemetry tracing and provides a span helper.
//
// NewTracer is the entry point: it builds a tracer provider ("otlp", "stdout",
// or "no-op"), registers it globally, and wires propagators. Instrument code
// with Tracer(ctx), which derives the span name from its caller via
// runtime.Caller — so call it directly at the top of a function; wrapping it in
// a helper breaks span naming.
//
// The OTLP exporter is HTTP: point OtelEndpoint at the collector's HTTP port
// (default :4318), not the gRPC port :4317.
package otel
