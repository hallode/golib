package email

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"gopkg.in/gomail.v2"
)

func TestNew_Validation(t *testing.T) {
	valid := Config{
		Host:     "smtp.example.com",
		Port:     587,
		Username: "user@example.com",
		Password: "secret",
	}

	client, err := New(valid)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	if client.from != valid.Username {
		t.Fatalf("from = %q, want %q", client.from, valid.Username)
	}

	tests := []struct {
		name string
		cfg  Config
	}{
		{name: "missing host", cfg: Config{Username: "a@b.com"}},
		{name: "missing username", cfg: Config{Host: "smtp.example.com"}},
		{name: "invalid from", cfg: Config{Host: "smtp.example.com", Username: "user@example.com", From: "not-an-email"}},
		{name: "invalid port", cfg: Config{Host: "smtp.example.com", Username: "user@example.com", Port: 70000}},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := New(tc.cfg); err == nil {
				t.Fatal("expected validation error")
			}
		})
	}
}

func TestNew_DefaultPort(t *testing.T) {
	client, err := New(Config{
		Host:     "smtp.example.com",
		Username: "user@example.com",
		Password: "secret",
	})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	if client.dialer.Port != defaultPort {
		t.Fatalf("dialer.Port = %d, want %d", client.dialer.Port, defaultPort)
	}
}

func TestMessage_Validation(t *testing.T) {
	client := testClient(t)

	t.Run("missing recipient", func(t *testing.T) {
		msg := client.NewMessage().Text("hello")
		if err := client.Send(msg); err == nil || !strings.Contains(err.Error(), "recipient") {
			t.Fatalf("Send() = %v", err)
		}
	})

	t.Run("missing body", func(t *testing.T) {
		msg := client.NewMessage().To("to@example.com")
		if err := client.Send(msg); err == nil || !strings.Contains(err.Error(), "body") {
			t.Fatalf("Send() = %v", err)
		}
	})
}

func TestMessage_SendPlainAndHTML(t *testing.T) {
	var sent []*gomail.Message
	client := testClientWithSender(t, func(msgs ...*gomail.Message) error {
		sent = append(sent, msgs...)
		return nil
	})

	err := client.Send(
		client.NewMessage().
			To("alice@example.com").
			Cc("cc@example.com").
			Bcc("bcc@example.com").
			ReplyTo("reply@example.com").
			Subject("Hello").
			Text("plain body").
			HTML("<p>html body</p>"),
	)
	if err != nil {
		t.Fatalf("Send() error = %v", err)
	}
	if len(sent) != 1 {
		t.Fatalf("sent %d messages, want 1", len(sent))
	}

	msg := sent[0]
	if got := msg.GetHeader(headerTo); len(got) != 1 || got[0] != "alice@example.com" {
		t.Fatalf("To header = %v", got)
	}
	if got := msg.GetHeader(headerSubject); len(got) != 1 || got[0] != "Hello" {
		t.Fatalf("Subject header = %v", got)
	}
}

func TestMessage_AttachBytes(t *testing.T) {
	var sent []*gomail.Message
	client := testClientWithSender(t, func(msgs ...*gomail.Message) error {
		sent = append(sent, msgs...)
		return nil
	})

	err := client.Send(
		client.NewMessage().
			To("alice@example.com").
			Text("see attachment").
			AttachBytes("report.pdf", []byte("%PDF-1.4"), "application/pdf"),
	)
	if err != nil {
		t.Fatalf("Send() error = %v", err)
	}
	if len(sent) != 1 {
		t.Fatalf("sent %d messages, want 1", len(sent))
	}
}

func TestClient_SendMany(t *testing.T) {
	var sent int
	client := testClientWithSender(t, func(msgs ...*gomail.Message) error {
		sent = len(msgs)
		return nil
	})

	err := client.SendMany(context.Background(),
		client.NewMessage().To("a@example.com").Text("one"),
		client.NewMessage().To("b@example.com").HTML("<p>two</p>"),
	)
	if err != nil {
		t.Fatalf("SendMany() error = %v", err)
	}
	if sent != 2 {
		t.Fatalf("sent %d messages, want 2", sent)
	}
}

func TestClient_SendContext_Cancelled(t *testing.T) {
	client := testClientWithSender(t, func(msgs ...*gomail.Message) error {
		time.Sleep(200 * time.Millisecond)
		return nil
	})

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()

	err := client.SendContext(ctx,
		client.NewMessage().To("a@example.com").Text("slow"),
	)
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("SendContext() = %v, want context.DeadlineExceeded", err)
	}
}

func TestMessage_FromOverride(t *testing.T) {
	var sent []*gomail.Message
	client := testClientWithSender(t, func(msgs ...*gomail.Message) error {
		sent = append(sent, msgs...)
		return nil
	})

	err := client.Send(
		client.NewMessage().
			From("noreply@example.com", "Example App").
			To("user@example.com").
			Text("hi"),
	)
	if err != nil {
		t.Fatalf("Send() error = %v", err)
	}
	from := sent[0].GetHeader(headerFrom)
	if len(from) != 1 || !strings.Contains(from[0], "noreply@example.com") {
		t.Fatalf("From header = %v", from)
	}
}

func testClient(t *testing.T) *Client {
	t.Helper()
	client, err := New(Config{
		Host:     "smtp.example.com",
		Username: "sender@example.com",
		Password: "secret",
	})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	return client
}

func testClientWithSender(t *testing.T, sendFn func(...*gomail.Message) error) *Client {
	t.Helper()
	client := testClient(t)
	client.sendFn = sendFn
	return client
}
