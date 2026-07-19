package log_test

import (
	"fmt"

	"github.com/hallode/golib/log"
)

// Sanitize redacts secret-looking values before they reach a log sink.
func ExampleSanitize() {
	fmt.Println(log.Sanitize("user=ada password=s3cr3t token=abc123"))
	// Output: user=ada ***PASSWORD*** ***TOKEN***
}

// ExampleNewWithConfig initializes the global logger once at startup.
func ExampleNewWithConfig() {
	log.NewWithConfig(log.Config{
		ServiceName:   "orders",
		EnableTraceID: true, // inject OpenTelemetry trace_id (pairs with golib/otel)
	})
	log.Info("service started")
}
