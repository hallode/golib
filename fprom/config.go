// Package fprom is a Prometheus metrics middleware for Fiber v3: a request
// counter and duration histogram plus a /metrics endpoint. Wiring is two steps —
// call Register(app) to expose /metrics and app.Use(m.Middleware) to record
// requests.
package fprom

import (
	"github.com/gofiber/fiber/v3"
)

type Config struct {
	Next        func(fiber.Ctx) bool
	MetricPath  string
	ServiceName string
	Namespace   string
	Subsystem   string
}

var ConfigDefault = Config{
	Next:       nil,
	MetricPath: "/metrics",
}

func configDefault(config ...Config) Config {
	if len(config) < 1 {
		return ConfigDefault
	}

	cfg := config[0]

	if cfg.MetricPath == "" {
		cfg.MetricPath = ConfigDefault.MetricPath
	}

	return cfg
}
