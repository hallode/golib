package otel

import (
	"context"
	"fmt"
	"log"
	"runtime"
	"strings"
	"sync"

	"go.opentelemetry.io/contrib/propagators/b3"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/propagation"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"
)

var (
	tracer             trace.Tracer
	moduleName         string
	initPropagatorOnce sync.Once
)

type TracerConfig struct {
	// Name is the service name (resource attribute service.name and the
	// module prefix stripped from Tracer(ctx) span names).
	Name string `json:"name"`
	// Tracer selects the exporter: "" or "no-op" (local SDK, nothing
	// exported), "stdout", or "otlp".
	Tracer string `json:"tracer"`
	// OtelEndpoint is the OTLP HTTP collector endpoint (host:port or URL).
	OtelEndpoint     string  `json:"otel_endpoint"`
	TracerSampleRate float64 `json:"tracer_sample_rate"`
	// Environment sets the deployment.environment resource attribute.
	// Empty falls back to $OTEL_ENVIRONMENT, then $SCENV, then "local".
	Environment string `json:"environment"`
	// Attributes are extra resource attributes appended to the defaults.
	Attributes []attribute.KeyValue `json:"-"`
	// Propagators overrides context propagation. Nil keeps the default:
	// W3C TraceContext + Baggage + B3 (multiple headers).
	Propagators []propagation.TextMapPropagator `json:"-"`
}

func NewTracer(config *TracerConfig) (trace.TracerProvider, error) {
	var tp trace.TracerProvider
	var err error

	if config.Tracer == "" {
		sdkTP := sdktrace.NewTracerProvider(
			sdktrace.WithSampler(sdktrace.AlwaysSample()),
		)
		otel.SetTracerProvider(sdkTP)
		tp = sdkTP
	} else {
		tp, err = NewProvider(config.Tracer, config.Name, config)
		if err != nil {
			return nil, fmt.Errorf("failed to create tracer provider: %w", err)
		}
	}

	initPropagatorOnce.Do(func() {
		propagators := config.Propagators
		if propagators == nil {
			propagators = []propagation.TextMapPropagator{
				propagation.TraceContext{},
				propagation.Baggage{},
				b3.New(b3.WithInjectEncoding(b3.B3MultipleHeader)),
			}
		}
		otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(propagators...))
	})

	moduleName = config.Name
	tracer = otel.Tracer(config.Name)

	logTracerInitialized(config)
	return tp, nil
}

func logTracerInitialized(c *TracerConfig) {
	switch c.Tracer {
	case "", ExporterNoop:
		log.Printf("otel: tracer ready service=%q (local SDK, spans not exported)", c.Name)
	case ExporterStdout:
		log.Printf("otel: tracer ready service=%q exporter=stdout", c.Name)
	default:
		log.Printf("otel: tracer ready service=%q exporter=otlp-http endpoint=%q", c.Name, c.OtelEndpoint)
	}
}

// unitTestCtxKey marks a context created by ContextWithUnitTest.
type unitTestCtxKey struct{}

// ContextWithUnitTest marks ctx so Tracer returns the original context
// instead of the span context (keeps unit tests deterministic).
func ContextWithUnitTest(ctx context.Context) context.Context {
	return context.WithValue(ctx, unitTestCtxKey{}, true)
}

func isUnitTest(ctx context.Context) bool {
	v, ok := ctx.Value(unitTestCtxKey{}).(bool)
	return ok && v
}

// Tracer starts a span named after the calling function. Call it directly at
// the top of the function to instrument — wrapping it breaks name derivation.
func Tracer(ctx context.Context) (context.Context, trace.Span) {
	pc, _, _, _ := runtime.Caller(1)
	f := runtime.FuncForPC(pc).Name()
	if moduleName != "" {
		f = strings.TrimPrefix(f, moduleName+"/")
	}
	replacer := strings.NewReplacer("(", "", ")", "", "*", "")
	operation := replacer.Replace(f)

	if tracer == nil {
		tracer = otel.Tracer("")
	}

	tracerCtx, span := tracer.Start(ctx, operation)
	if isUnitTest(ctx) {
		return ctx, span
	}
	return tracerCtx, span
}

// GetContextBackground returns context.Background() with the span from ctx attached (for goroutines).
func GetContextBackground(ctx context.Context) context.Context {
	return trace.ContextWithSpan(context.Background(), trace.SpanFromContext(ctx))
}
