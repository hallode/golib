package custerr_test

import (
	"errors"
	"net/http"
	"strings"
	"testing"

	"github.com/hallode/golib/v2/custerr"
)

func TestWrap_Nil(t *testing.T) {
	if got := custerr.Wrap(nil); got != nil {
		t.Fatalf("Wrap(nil) = %v, want nil", got)
	}
	if got := custerr.Wrapf(nil, "ctx"); got != nil {
		t.Fatalf("Wrapf(nil) = %v, want nil", got)
	}
}

func TestWrap_CapturesSource(t *testing.T) {
	base := errors.New("base")
	wrapped := custerr.Wrap(base)

	var sourcer custerr.Sourcer
	if !errors.As(wrapped, &sourcer) {
		t.Fatal("wrapped error should implement Sourcer")
	}
	src := sourcer.Source()
	if src == "" {
		t.Fatal("expected non-empty source")
	}
	if !strings.Contains(src, "custerr_test.go") {
		t.Fatalf("source %q should reference test file", src)
	}
	if !errors.Is(wrapped, base) {
		t.Fatal("wrapped error should unwrap to base")
	}
}

func TestWrapf_Message(t *testing.T) {
	base := errors.New("base")
	wrapped := custerr.Wrapf(base, "operation %s failed", "save")

	if got := wrapped.Error(); !strings.Contains(got, "operation save failed") {
		t.Fatalf("Error() = %q, want context message", got)
	}

	var msg custerr.Messager
	if !errors.As(wrapped, &msg) {
		t.Fatal("wrapped error should implement Messager")
	}
	if msg.Message() != "operation save failed" {
		t.Fatalf("Message() = %q", msg.Message())
	}
}

func TestCuster_UnwrapAndSource(t *testing.T) {
	base := errors.New("validation failed")
	err := custerr.BadRequest(base).WithCode(custerr.CodeValidationBadRequest)

	if err.StatusCode != http.StatusBadRequest {
		t.Fatalf("StatusCode = %d", err.StatusCode)
	}
	if err.Code != custerr.CodeValidationBadRequest {
		t.Fatalf("Code = %d", err.Code)
	}
	if !errors.Is(err, base) {
		t.Fatal("Custer should unwrap to base error")
	}
	if err.Source() == "" {
		t.Fatal("expected captured source on Custer")
	}
}

func TestBusinessCodeByStatus(t *testing.T) {
	tests := []struct {
		status int
		want   int
	}{
		{http.StatusUnauthorized, custerr.CodeAuthUnauthorized},
		{http.StatusNotFound, custerr.CodeValidationNotFound},
		{http.StatusBadGateway, custerr.CodeIntegrationExternal},
		{http.StatusInternalServerError, custerr.CodeInternalUnknown},
		{http.StatusTeapot, custerr.CodeValidationBadRequest},
	}

	for _, tc := range tests {
		if got := custerr.BusinessCodeByStatus(tc.status); got != tc.want {
			t.Errorf("BusinessCodeByStatus(%d) = %d, want %d", tc.status, got, tc.want)
		}
	}
}

func TestRegisterStatusFallback(t *testing.T) {
	const custom = 9999999
	custerr.RegisterStatusFallback(http.StatusTeapot, custom)
	if got := custerr.BusinessCodeByStatus(http.StatusTeapot); got != custom {
		t.Fatalf("RegisterStatusFallback() = %d, want %d", got, custom)
	}
}

func TestStatusText(t *testing.T) {
	if got := custerr.StatusText(http.StatusNotFound); got != "Not Found" {
		t.Fatalf("StatusText(404) = %q", got)
	}
	if got := custerr.StatusText(999); got != "Oops something went wrong" {
		t.Fatalf("StatusText(999) = %q", got)
	}
}

func TestNonRetryable(t *testing.T) {
	base := errors.New("bad payload")
	err := custerr.NewNonRetryable(base)

	if !custerr.IsNonRetryable(err) {
		t.Fatal("expected IsNonRetryable true")
	}
	if !errors.Is(err, base) {
		t.Fatal("NonRetryable should unwrap to base")
	}

	var sourcer custerr.Sourcer
	if !errors.As(err, &sourcer) || sourcer.Source() == "" {
		t.Fatal("NonRetryable should carry source from Wrap")
	}
}

func TestSetAppModule(t *testing.T) {
	t.Cleanup(func() { custerr.SetAppModule("") })

	// The package directory "custerr/" is always part of this file's path
	// (independent of checkout location), so it is stripped to a
	// module-relative form: "/custerr/custerr_test.go:NN".
	custerr.SetAppModule("custerr/")
	err := custerr.Wrap(errors.New("x"))

	var sourcer custerr.Sourcer
	if !errors.As(err, &sourcer) {
		t.Fatal("expected Sourcer")
	}
	if !strings.HasPrefix(sourcer.Source(), "/custerr/") {
		t.Fatalf("source %q should be rewritten to a module-relative path", sourcer.Source())
	}
}

func TestSetAppModule_DefaultKeepsRawPath(t *testing.T) {
	// With no app module set (the default), the captured source keeps its raw
	// absolute path rather than a fabricated prefix.
	err := custerr.Wrap(errors.New("x"))

	var sourcer custerr.Sourcer
	if !errors.As(err, &sourcer) {
		t.Fatal("expected Sourcer")
	}
	if !strings.HasPrefix(sourcer.Source(), "/") || strings.HasPrefix(sourcer.Source(), "//") {
		t.Fatalf("source %q should be a clean raw absolute path", sourcer.Source())
	}
}
