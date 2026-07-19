// Package sql manages a PostgreSQL connection pool built on pgx/v5. New parses a
// DSN (pgx URL or keyword form), applies pool settings, connects and pings; run
// queries through the exposed Pool. It handles lifecycle only — no query helpers.
package sql

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	defaultMaxConns          int32 = 30
	defaultMinConns          int32 = 2
	defaultMaxConnLifetime         = 30 * time.Minute
	defaultMaxConnIdleTime         = 5 * time.Minute
	defaultHealthCheckPeriod       = 1 * time.Minute
	defaultConnectTimeout          = 10 * time.Second
)

var ErrEmptyDSN = errors.New("sql: dsn is required")

type (
	Options struct {
		DSN               string
		MaxConns          int32
		MinConns          int32
		MaxConnLifetime   time.Duration
		MaxConnIdleTime   time.Duration
		HealthCheckPeriod time.Duration
		ConnectTimeout    time.Duration
	}

	SQL struct {
		Pool *pgxpool.Pool
	}
)

func (opt *Options) init() {
	if opt.MaxConns == 0 {
		opt.MaxConns = defaultMaxConns
	}
	if opt.MinConns == 0 {
		opt.MinConns = defaultMinConns
	}
	if opt.MaxConnLifetime == 0 {
		opt.MaxConnLifetime = defaultMaxConnLifetime
	}
	if opt.MaxConnIdleTime == 0 {
		opt.MaxConnIdleTime = defaultMaxConnIdleTime
	}
	if opt.HealthCheckPeriod == 0 {
		opt.HealthCheckPeriod = defaultHealthCheckPeriod
	}
	if opt.ConnectTimeout == 0 {
		opt.ConnectTimeout = defaultConnectTimeout
	}
}

func (opt *Options) validate() error {
	if strings.TrimSpace(opt.DSN) == "" {
		return ErrEmptyDSN
	}
	if opt.MinConns > opt.MaxConns {
		return fmt.Errorf("sql: min conns (%d) cannot exceed max conns (%d)", opt.MinConns, opt.MaxConns)
	}
	return nil
}

// New creates a PostgreSQL connection pool backed by pgx. The caller must call Close on shutdown.
func New(opt *Options) (*SQL, error) {
	if opt == nil {
		return nil, errors.New("sql: options is nil")
	}

	opt.init()
	if err := opt.validate(); err != nil {
		return nil, err
	}

	pool, err := connect(opt)
	if err != nil {
		return nil, err
	}

	return &SQL{Pool: pool}, nil
}

func connect(opt *Options) (*pgxpool.Pool, error) {
	cfg, err := pgxpool.ParseConfig(opt.DSN)
	if err != nil {
		return nil, fmt.Errorf("sql: parse dsn: %w", err)
	}

	cfg.MaxConns = opt.MaxConns
	cfg.MinConns = opt.MinConns
	cfg.MaxConnLifetime = opt.MaxConnLifetime
	cfg.MaxConnIdleTime = opt.MaxConnIdleTime
	cfg.HealthCheckPeriod = opt.HealthCheckPeriod

	ctx, cancel := context.WithTimeout(context.Background(), opt.ConnectTimeout)
	defer cancel()

	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("sql: connect: %w", err)
	}

	if err = pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("sql: ping: %w", err)
	}

	return pool, nil
}

// Ping checks whether the database is reachable.
func (s *SQL) Ping(ctx context.Context) error {
	if s == nil || s.Pool == nil {
		return errors.New("sql: pool is not connected")
	}
	if ctx == nil {
		ctx = context.Background()
	}
	return s.Pool.Ping(ctx)
}

// Close closes the connection pool and releases all resources.
func (s *SQL) Close() {
	if s == nil || s.Pool == nil {
		return
	}
	s.Pool.Close()
	s.Pool = nil
}
