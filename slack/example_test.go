package slack_test

import (
	"context"
	"fmt"

	"github.com/hallode/golib/v2/slack"
)

func ExampleBuildAlert() {
	client := slack.New("https://hooks.slack.com/services/T000/B000/XXXX", "#alerts")
	slack.InitAlertWorker(client, noopLogger{})
	defer slack.StopAlertWorker()

	err := slack.BuildAlert(slack.SeverityError, "orders", "failed to charge card").
		WithPath("/v1/checkout").
		WithStatusCode(500).
		WithTraceID("abc123").
		Send(context.Background())
	if err != nil {
		fmt.Println(err)
	}
}

// noopLogger satisfies the logger dependency of the alert worker.
type noopLogger struct{}

func (noopLogger) Errorf(string, ...any) {}
func (noopLogger) Infof(string, ...any)  {}
func (noopLogger) Warnf(string, ...any)  {}
