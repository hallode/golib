package fprom_test

import (
	"github.com/gofiber/fiber/v3"

	"github.com/hallode/golib/v2/fprom"
)

func ExampleNew() {
	app := fiber.New()

	m := fprom.New(fprom.Config{ServiceName: "orders", Namespace: "myapp"})
	m.Register(app)       // exposes GET /metrics
	app.Use(m.Middleware) // records every request
}
