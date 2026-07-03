package fprom

import (
	"fmt"
	"strings"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/adaptor"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

type Middleware struct {
	config       Config
	registry     *prometheus.Registry
	totalRequest *prometheus.CounterVec
	duration     *prometheus.HistogramVec
}

func New(config ...Config) *Middleware {
	cfg := configDefault(config...)

	return createDefaultMetrics(cfg)
}

func createDefaultMetrics(config Config) *Middleware {
	constLabels := make(prometheus.Labels)
	if config.ServiceName != "" {
		constLabels["service"] = config.ServiceName
	}

	reg := prometheus.NewRegistry()

	totalRequest := prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name:        "http_requests_total",
			Namespace:   strings.ReplaceAll(config.Namespace, "-", ""),
			Subsystem:   strings.ReplaceAll(config.Subsystem, "-", ""),
			Help:        "Number of HTTP requests",
			ConstLabels: constLabels,
		},
		[]string{"path", "method", "code"},
	)

	duration := prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Name:        "http_request_duration_seconds",
		Namespace:   strings.ReplaceAll(config.Namespace, "-", ""),
		Subsystem:   strings.ReplaceAll(config.Subsystem, "-", ""),
		Help:        "Duration of HTTP requests",
		ConstLabels: constLabels,
	}, []string{"path", "method"})

	reg.MustRegister(
		collectors.NewGoCollector(),
		collectors.NewProcessCollector(collectors.ProcessCollectorOpts{}),
		totalRequest,
		duration,
	)

	return &Middleware{
		config:       config,
		registry:     reg,
		totalRequest: totalRequest,
		duration:     duration,
	}
}

func (m *Middleware) Register(app *fiber.App) {
	app.Get(m.config.MetricPath, adaptor.HTTPHandler(promhttp.HandlerFor(m.registry, promhttp.HandlerOpts{})))
}

func (m *Middleware) Middleware(c fiber.Ctx) error {
	if c.Path() == m.config.MetricPath {
		return c.Next()
	}

	if m.config.Next != nil && m.config.Next(c) {
		return c.Next()
	}

	start := time.Now()
	err := c.Next()

	r := c.Route()
	statusCode := fmt.Sprintf("%v", c.Response().StatusCode())
	elapsed := float64(time.Since(start)) / float64(time.Second)

	m.totalRequest.WithLabelValues(r.Path, r.Method, statusCode).Inc()
	m.duration.WithLabelValues(r.Path, r.Method).Observe(elapsed)

	return err
}
