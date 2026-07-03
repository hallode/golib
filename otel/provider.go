package otel

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/url"
	"os"
	"strings"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	stdout "go.opentelemetry.io/otel/exporters/stdout/stdouttrace"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
	"go.opentelemetry.io/otel/trace"
)

// Exporter kinds accepted by NewProvider / TracerConfig.Tracer.
const (
	ExporterNoop   = "no-op"
	ExporterStdout = "stdout"
	ExporterOTLP   = "otlp"
)

func validateTracerSampleRate(rate float64) error {
	if rate == 0 {
		return nil
	}
	if rate < 0 || rate > 1 {
		return fmt.Errorf("tracer_sample_rate must be in (0,1] or omitted for 100%%, got %g", rate)
	}
	return nil
}

func NewProvider(kind, name string, cfg *TracerConfig) (trace.TracerProvider, error) {
	if kind == "" || kind == ExporterNoop {
		return nil, nil
	}

	if cfg == nil {
		return nil, errors.New("missing tracer configuration")
	}

	if err := validateTracerSampleRate(cfg.TracerSampleRate); err != nil {
		return nil, err
	}

	var exporter sdktrace.SpanExporter
	var err error

	switch kind {
	case ExporterStdout:
		exporter, err = stdout.New(stdout.WithPrettyPrint())
		if err != nil {
			return nil, fmt.Errorf("creating stdout exporter: %w", err)
		}
	case ExporterOTLP:
		if strings.TrimSpace(cfg.OtelEndpoint) == "" {
			return nil, errors.New("otel_endpoint is required for the otlp exporter")
		}
		exporter, err = newOTLPHTTPExporter(cfg.OtelEndpoint)
		if err != nil {
			return nil, fmt.Errorf("otlp http exporter: %w", err)
		}
	default:
		return nil, fmt.Errorf("unsupported tracer kind: %q (supported: otlp, stdout, no-op)", kind)
	}

	sampleRate := 1.0
	if cfg.TracerSampleRate > 0 {
		sampleRate = cfg.TracerSampleRate
	}

	attrs := append([]attribute.KeyValue{
		semconv.ServiceNameKey.String(name),
		semconv.DeploymentEnvironmentKey.String(cfg.environment()),
	}, cfg.Attributes...)

	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithSampler(sdktrace.TraceIDRatioBased(sampleRate)),
		sdktrace.WithResource(resource.NewWithAttributes(semconv.SchemaURL, attrs...)),
	)

	otel.SetTracerProvider(tp)
	return tp, nil
}

func newOTLPHTTPExporter(raw string) (sdktrace.SpanExporter, error) {
	hostPort, insecure, err := resolveOTLPHTTPHostPort(raw)
	if err != nil {
		return nil, err
	}
	opts := []otlptracehttp.Option{otlptracehttp.WithEndpoint(hostPort)}
	if insecure {
		opts = append(opts, otlptracehttp.WithInsecure())
	}
	return otlptrace.New(context.Background(), otlptracehttp.NewClient(opts...))
}

func resolveOTLPHTTPHostPort(raw string) (hostPort string, insecure bool, err error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", false, errors.New("empty otel_endpoint")
	}

	if strings.Contains(raw, "://") {
		u, err := url.Parse(raw)
		if err != nil {
			return "", false, err
		}
		host := u.Hostname()
		if host == "" {
			return "", false, fmt.Errorf("missing host in %q", raw)
		}
		port := u.Port()
		if port == "" {
			if u.Scheme == "https" {
				port = "443"
			} else {
				port = "80"
			}
		}
		insecure = u.Scheme != "https"

		if port == "4317" {
			return "", false, fmt.Errorf("port 4317 is OTLP gRPC; this package exports OTLP HTTP only — use :4318 (e.g. https://%s:4318)", host)
		}
		return net.JoinHostPort(host, port), insecure, nil
	}

	host, port, splitErr := net.SplitHostPort(raw)
	if splitErr != nil {
		if strings.Contains(raw, ":") {
			return "", false, fmt.Errorf("invalid otel_endpoint %q: %w", raw, splitErr)
		}
		return net.JoinHostPort(raw, "4318"), true, nil
	}

	if port == "4317" {
		return "", false, fmt.Errorf("otel_endpoint %q uses gRPC port 4317; use OTLP HTTP :4318", raw)
	}
	return net.JoinHostPort(host, port), true, nil
}

// environment resolves deployment.environment: explicit config first, then
// the OTEL_ENVIRONMENT and SCENV env vars, then "local".
func (c *TracerConfig) environment() string {
	if c.Environment != "" {
		return c.Environment
	}
	for _, key := range []string{"OTEL_ENVIRONMENT", "SCENV"} {
		if env := os.Getenv(key); env != "" {
			return env
		}
	}
	return "local"
}
