package slack

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/hallode/golib/v2/custerr"
)

func TestHasMeaningfulContent(t *testing.T) {
	tests := []struct {
		input string
		want  bool
	}{
		{"", false},
		{"  ", false},
		{"{}", false},
		{"[]", false},
		{"null", false},
		{`{"id":1}`, true},
		{"page=1", true},
	}

	for _, tc := range tests {
		if got := hasMeaningfulContent(tc.input); got != tc.want {
			t.Errorf("hasMeaningfulContent(%q) = %v, want %v", tc.input, got, tc.want)
		}
	}
}

func TestExtractErrorMessage(t *testing.T) {
	if got := ExtractErrorMessage(nil); got != "" {
		t.Fatalf("ExtractErrorMessage(nil) = %q", got)
	}

	long := strings.Repeat("x", 600)
	if got := ExtractErrorMessage(errors.New(long)); len(got) > 503 {
		t.Fatalf("ExtractErrorMessage() should truncate, len=%d", len(got))
	}

	if got := ExtractErrorMessage(errors.New("internal server error")); got != "An unexpected error occurred" {
		t.Fatalf("ExtractErrorMessage() = %q", got)
	}
}

func TestExtractErrorSource(t *testing.T) {
	if got := ExtractErrorSource(nil); got != "" {
		t.Fatalf("ExtractErrorSource(nil) = %q", got)
	}

	inner := custerr.Wrap(errors.New("root"))
	outer := custerr.Wrapf(inner, "outer")

	got := ExtractErrorSource(outer)
	if got == "" {
		t.Fatal("ExtractErrorSource() should return source from error chain")
	}
	if !strings.Contains(got, "alert_test.go") {
		t.Fatalf("ExtractErrorSource() = %q", got)
	}
}

func TestTruncateString(t *testing.T) {
	if got := truncateString("short", 10); got != "short" {
		t.Fatalf("truncateString() = %q", got)
	}
	got := truncateString(strings.Repeat("a", 20), 10)
	if !strings.HasSuffix(got, "... (truncated)") {
		t.Fatalf("truncateString() = %q", got)
	}
}

func TestTruncateStackTrace(t *testing.T) {
	lines := make([]string, 50)
	for i := range lines {
		lines[i] = "line"
	}
	stack := strings.Join(lines, "\n")

	got := truncateStackTrace(stack, 30)
	if !strings.Contains(got, "... (truncated, total 50 lines)") {
		t.Fatalf("truncateStackTrace() = %q", got)
	}
}

func TestRateLimiter(t *testing.T) {
	rl := newRateLimiter(2, time.Minute)

	if !rl.allow() || !rl.allow() {
		t.Fatal("first two requests should be allowed")
	}
	if rl.allow() {
		t.Fatal("third request should be denied before refill")
	}
}
