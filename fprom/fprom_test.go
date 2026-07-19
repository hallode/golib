package fprom_test

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gofiber/fiber/v3"

	"github.com/hallode/golib/v2/fprom"
)

func TestMiddleware_RecordsAndExposesMetrics(t *testing.T) {
	app := fiber.New()

	m := fprom.New(fprom.Config{ServiceName: "test", Namespace: "golibtest"})
	m.Register(app)
	app.Use(m.Middleware)
	app.Get("/ping", func(c fiber.Ctx) error { return c.SendString("pong") })

	// Drive one request so a counter sample exists.
	resp, err := app.Test(httptest.NewRequest(http.MethodGet, "/ping", nil))
	if err != nil {
		t.Fatalf("Test(/ping): %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("/ping status = %d, want 200", resp.StatusCode)
	}

	// The /metrics endpoint should expose the namespaced counter.
	resp, err = app.Test(httptest.NewRequest(http.MethodGet, "/metrics", nil))
	if err != nil {
		t.Fatalf("Test(/metrics): %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("/metrics status = %d, want 200", resp.StatusCode)
	}

	body, _ := io.ReadAll(resp.Body)
	if !strings.Contains(string(body), "golibtest_http_requests_total") {
		t.Fatalf("/metrics missing namespaced counter, got:\n%s", string(body))
	}
}

func TestNew_DefaultMetricPath(t *testing.T) {
	m := fprom.New()
	app := fiber.New()
	m.Register(app)

	resp, err := app.Test(httptest.NewRequest(http.MethodGet, "/metrics", nil))
	if err != nil {
		t.Fatalf("Test(/metrics): %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("default /metrics status = %d, want 200", resp.StatusCode)
	}
}
