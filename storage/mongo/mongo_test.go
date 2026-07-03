package mongo

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestOptionsInit_Defaults(t *testing.T) {
	opt := &Options{}
	opt.init()

	if opt.AppName != "Default" {
		t.Fatalf("AppName = %q, want Default", opt.AppName)
	}
	if opt.ConnectTimeout != 10*time.Second {
		t.Fatalf("ConnectTimeout = %v, want 10s", opt.ConnectTimeout)
	}
	if opt.PingTimeout != 2*time.Second {
		t.Fatalf("PingTimeout = %v, want 2s", opt.PingTimeout)
	}
}

func TestOptionsValidate(t *testing.T) {
	tests := []struct {
		name string
		opt  Options
		want error
	}{
		{
			name: "missing uri",
			opt:  Options{DB: "app"},
			want: ErrEmptyURI,
		},
		{
			name: "missing db",
			opt:  Options{URI: "mongodb://localhost:27017"},
			want: ErrEmptyDB,
		},
		{
			name: "valid",
			opt:  Options{URI: "mongodb://localhost:27017", DB: "app"},
			want: nil,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			tc.opt.init()
			err := tc.opt.validate()
			if !errors.Is(err, tc.want) {
				t.Fatalf("validate() = %v, want %v", err, tc.want)
			}
		})
	}
}

func TestNew_NilOptions(t *testing.T) {
	if _, err := New(nil); err == nil {
		t.Fatal("expected error for nil options")
	}
}

func TestNew_ValidationBeforeConnect(t *testing.T) {
	_, err := New(&Options{})
	if !errors.Is(err, ErrEmptyURI) {
		t.Fatalf("New() = %v, want ErrEmptyURI", err)
	}
}

func TestNew_WithOpenTelemetry(t *testing.T) {
	// Enabled path should not fail when no TracerProvider is configured (OTel no-op).
	_, err := New(&Options{
		URI:                 "mongodb://127.0.0.1:1/app?connect_timeout=1",
		DB:                  "app",
		ConnectTimeout:      2 * time.Second,
		EnableOpenTelemetry: true,
	})
	if err == nil {
		t.Fatal("expected connection error")
	}
}

func TestNew_ConnectionFailure(t *testing.T) {
	_, err := New(&Options{
		URI: "mongodb://127.0.0.1:1",
		DB:  "app",
	})
	if err == nil {
		t.Fatal("expected connection error")
	}
}

func TestMongo_CloseAndPingGuards(t *testing.T) {
	var m *Mongo
	if err := m.Close(context.Background()); err != nil {
		t.Fatalf("Close(nil receiver) = %v", err)
	}
	if err := m.Ping(context.Background()); err == nil {
		t.Fatal("Ping(nil receiver) should fail")
	}
	if m.Client() != nil {
		t.Fatal("Client(nil receiver) should be nil")
	}
}

func TestMongo_CloseClearsState(t *testing.T) {
	m := &Mongo{
		client: nil,
		DB:     nil,
	}
	if err := m.Close(context.Background()); err != nil {
		t.Fatalf("Close() = %v", err)
	}
}
