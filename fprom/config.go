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
