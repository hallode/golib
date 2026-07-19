// Package mongo manages a MongoDB connection on mongo-driver/v2. New connects and
// pings; use the exposed DB for collection operations and Client() for
// transactions. EnableOpenTelemetry requires a global OTel provider (golib/otel).
package mongo

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
	"go.mongodb.org/mongo-driver/v2/mongo/readpref"
	"go.opentelemetry.io/contrib/instrumentation/go.mongodb.org/mongo-driver/v2/mongo/otelmongo"
)

const defaultCloseTimeout = 10 * time.Second

var (
	ErrEmptyURI = errors.New("mongo: uri is required")
	ErrEmptyDB  = errors.New("mongo: db is required")
)

type (
	Options struct {
		URI            string
		DB             string
		AppName        string
		ConnectTimeout time.Duration
		PingTimeout    time.Duration

		// EnableOpenTelemetry attaches otelmongo command monitoring to the client.
		// Default is false; set true when the service exports OpenTelemetry data.
		EnableOpenTelemetry bool
	}

	Mongo struct {
		DB *mongo.Database

		client *mongo.Client
	}
)

func (opt *Options) init() {
	if opt.AppName == "" {
		opt.AppName = "Default"
	}
	if opt.ConnectTimeout == 0 {
		opt.ConnectTimeout = 10 * time.Second
	}
	if opt.PingTimeout == 0 {
		opt.PingTimeout = 2 * time.Second
	}
}

func (opt *Options) validate() error {
	if strings.TrimSpace(opt.URI) == "" {
		return ErrEmptyURI
	}
	if strings.TrimSpace(opt.DB) == "" {
		return ErrEmptyDB
	}
	return nil
}

// New connects to MongoDB, verifies connectivity with Ping, and returns a handle
// to the configured database. The caller must call Close on shutdown.
func New(opt *Options) (*Mongo, error) {
	if opt == nil {
		return nil, errors.New("mongo: options is nil")
	}

	opt.init()
	if err := opt.validate(); err != nil {
		return nil, err
	}

	client, err := connect(opt)
	if err != nil {
		return nil, err
	}

	return &Mongo{
		client: client,
		DB:     client.Database(opt.DB),
	}, nil
}

func connect(opt *Options) (*mongo.Client, error) {
	clientOpts := options.Client().
		ApplyURI(opt.URI).
		SetAppName(opt.AppName).
		SetConnectTimeout(opt.ConnectTimeout)

	if opt.EnableOpenTelemetry {
		clientOpts.SetMonitor(otelmongo.NewMonitor())
	}

	client, err := mongo.Connect(clientOpts)
	if err != nil {
		return nil, fmt.Errorf("mongo: connect: %w", err)
	}

	pingCtx, cancelPing := context.WithTimeout(context.Background(), opt.PingTimeout)
	defer cancelPing()

	if err = client.Ping(pingCtx, readpref.Primary()); err != nil {
		disconnectCtx, cancelDisconnect := context.WithTimeout(context.Background(), defaultCloseTimeout)
		_ = client.Disconnect(disconnectCtx)
		cancelDisconnect()
		return nil, fmt.Errorf("mongo: ping: %w", err)
	}

	return client, nil
}

// Client exposes the underlying client for transactions and multi-database access.
func (m *Mongo) Client() *mongo.Client {
	if m == nil {
		return nil
	}
	return m.client
}

// Ping checks whether the primary node is reachable.
func (m *Mongo) Ping(ctx context.Context) error {
	if m == nil || m.client == nil {
		return errors.New("mongo: client is not connected")
	}
	if ctx == nil {
		ctx = context.Background()
	}
	return m.client.Ping(ctx, readpref.Primary())
}

// Close disconnects the client and releases connection pool resources.
func (m *Mongo) Close(ctx context.Context) error {
	if m == nil || m.client == nil {
		return nil
	}

	if ctx == nil {
		ctx = context.Background()
	}
	if _, hasDeadline := ctx.Deadline(); !hasDeadline {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, defaultCloseTimeout)
		defer cancel()
	}

	err := m.client.Disconnect(ctx)
	m.client = nil
	m.DB = nil
	return err
}
