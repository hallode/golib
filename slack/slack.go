// Package slack posts messages to a Slack incoming webhook and provides an async
// AlertWorker that formats, rate-limits, and aggregates error alerts. Build
// alerts with the fluent BuildAlert(...).Send(ctx); a webhook URL is required.
package slack

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/hallode/golib/v2/json"

	"github.com/pkg/errors"
)

const defaultHTTPTimeout = 10 * time.Second

type Client struct {
	WebhookURL     string
	DefaultChannel string

	// HTTPClient is used to deliver webhooks. Defaults to a client with a
	// 10s timeout when nil.
	HTTPClient *http.Client
}

// New creates a slack webhook client. Consuming services pass values from
// their own config source.
func New(webhookURL, defaultChannel string) *Client {
	return &Client{
		WebhookURL:     webhookURL,
		DefaultChannel: defaultChannel,
	}
}

type message struct {
	Text    string `json:"text,omitempty"`
	Channel string `json:"channel,omitempty"`
}

func (c Client) httpClient() *http.Client {
	if c.HTTPClient != nil {
		return c.HTTPClient
	}
	return &http.Client{Timeout: defaultHTTPTimeout}
}

func (c Client) Send(ctx context.Context, text string) error {
	if c.WebhookURL == "" {
		return errors.New("slack webhook url is empty")
	}

	payload := message{
		Text:    text,
		Channel: c.DefaultChannel,
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed json.Marshal slack payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.WebhookURL, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("failed to build slack request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient().Do(req)
	if err != nil {
		return fmt.Errorf("failed to send slack message: network/context error: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusBadRequest {
		return fmt.Errorf("slack webhook returned status code: %d", resp.StatusCode)
	}
	return nil
}
