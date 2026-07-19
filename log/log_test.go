package log_test

import (
	"strings"
	"testing"

	"context"

	customlog "github.com/hallode/golib/v2/log"
)

func TestSanitize_RedactsSensitiveFields(t *testing.T) {
	tests := []struct {
		name  string
		input any
		want  []string
		not   []string
	}{
		{
			name:  "password in JSON",
			input: `{"username":"alice","password":"secret123"}`,
			want:  []string{"***PASSWORD***", "alice"},
			not:   []string{"secret123"},
		},
		{
			name:  "token query param",
			input: "token=abc123",
			want:  []string{"***TOKEN***"},
			not:   []string{"abc123"},
		},
		{
			name:  "struct marshaled to JSON",
			input: map[string]string{"api_key": "sk-live-abc"},
			want:  []string{"***API_KEY***"},
			not:   []string{"sk-live-abc"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := customlog.Sanitize(tc.input)
			for _, s := range tc.want {
				if !strings.Contains(got, s) {
					t.Errorf("Sanitize() = %q, want substring %q", got, s)
				}
			}
			for _, s := range tc.not {
				if strings.Contains(got, s) {
					t.Errorf("Sanitize() = %q, must not contain %q", got, s)
				}
			}
		})
	}
}

func TestSanitize_TruncatesLongOutput(t *testing.T) {
	long := strings.Repeat("x", 3000)
	got := customlog.Sanitize(long)
	if len(got) <= 2000 {
		t.Fatalf("expected truncated output, len=%d", len(got))
	}
	if !strings.Contains(got, "... (truncated)") {
		t.Fatalf("Sanitize() = %q, want truncation marker", got)
	}
}

func TestSanitize_UnmarshalableReturnsEmpty(t *testing.T) {
	ch := make(chan int)
	if got := customlog.Sanitize(ch); got != "" {
		t.Fatalf("Sanitize(channel) = %q, want empty string", got)
	}
}

func TestNewAndIsDebug(t *testing.T) {
	customlog.New("log-test")
	customlog.SetLevel("debug")
	if !customlog.IsDebug() {
		t.Fatal("IsDebug() should be true after SetLevel(debug)")
	}
}

func TestWithTraceID_OptIn(t *testing.T) {
	customlog.NewWithConfig(customlog.Config{
		ServiceName:   "log-test",
		EnableTraceID: false,
	})
	logger := customlog.With(context.Background())
	if logger == nil {
		t.Fatal("With() returned nil")
	}
}
