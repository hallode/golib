package sql

import (
	"errors"
	"testing"
	"time"
)

func TestOptionsInit_Defaults(t *testing.T) {
	opt := &Options{}
	opt.init()

	if opt.MaxConns != defaultMaxConns {
		t.Fatalf("MaxConns = %d, want %d", opt.MaxConns, defaultMaxConns)
	}
	if opt.MinConns != defaultMinConns {
		t.Fatalf("MinConns = %d, want %d", opt.MinConns, defaultMinConns)
	}
	if opt.MaxConnLifetime != defaultMaxConnLifetime {
		t.Fatalf("MaxConnLifetime = %v", opt.MaxConnLifetime)
	}
	if opt.MaxConnIdleTime != defaultMaxConnIdleTime {
		t.Fatalf("MaxConnIdleTime = %v", opt.MaxConnIdleTime)
	}
	if opt.HealthCheckPeriod != defaultHealthCheckPeriod {
		t.Fatalf("HealthCheckPeriod = %v", opt.HealthCheckPeriod)
	}
	if opt.ConnectTimeout != defaultConnectTimeout {
		t.Fatalf("ConnectTimeout = %v", opt.ConnectTimeout)
	}
}

func TestOptionsValidate(t *testing.T) {
	tests := []struct {
		name    string
		opt     Options
		wantErr error
	}{
		{
			name:    "missing dsn",
			opt:     Options{},
			wantErr: ErrEmptyDSN,
		},
		{
			name: "min greater than max",
			opt: Options{
				DSN:      "postgres://user:pass@localhost:5432/app",
				MinConns: 10,
				MaxConns: 5,
			},
			wantErr: errors.New("sql: min conns"),
		},
		{
			name: "valid",
			opt: Options{
				DSN: "postgres://user:pass@localhost:5432/app",
			},
			wantErr: nil,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			tc.opt.init()
			err := tc.opt.validate()
			if tc.wantErr == nil {
				if err != nil {
					t.Fatalf("validate() = %v", err)
				}
				return
			}
			if err == nil {
				t.Fatalf("validate() = nil, want error")
			}
			if tc.wantErr == ErrEmptyDSN && !errors.Is(err, ErrEmptyDSN) {
				t.Fatalf("validate() = %v, want ErrEmptyDSN", err)
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
	if !errors.Is(err, ErrEmptyDSN) {
		t.Fatalf("New() = %v, want ErrEmptyDSN", err)
	}
}

func TestNew_InvalidDSN(t *testing.T) {
	_, err := New(&Options{DSN: "://invalid"})
	if err == nil {
		t.Fatal("expected parse error")
	}
}

func TestNew_ConnectionFailure(t *testing.T) {
	_, err := New(&Options{
		DSN:            "postgres://postgres:postgres@127.0.0.1:1/app?connect_timeout=1",
		ConnectTimeout: 2 * time.Second,
	})
	if err == nil {
		t.Fatal("expected connection error")
	}
}

func TestSQL_CloseAndPingGuards(t *testing.T) {
	var s *SQL
	s.Close()
	if err := s.Ping(nil); err == nil {
		t.Fatal("Ping(nil receiver) should fail")
	}

	empty := &SQL{}
	empty.Close()
	if err := empty.Ping(nil); err == nil {
		t.Fatal("Ping on closed pool should fail")
	}
}

func TestOptionsInit_PreservesCustomValues(t *testing.T) {
	opt := &Options{
		MaxConns:          50,
		MinConns:          5,
		MaxConnLifetime:   time.Hour,
		MaxConnIdleTime:   10 * time.Minute,
		HealthCheckPeriod: 30 * time.Second,
		ConnectTimeout:    15 * time.Second,
	}
	opt.init()

	if opt.MaxConns != 50 || opt.MinConns != 5 {
		t.Fatalf("pool size overrides not preserved: max=%d min=%d", opt.MaxConns, opt.MinConns)
	}
}
