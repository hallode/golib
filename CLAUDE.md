# CLAUDE.md

Guidance for Claude Code when working in this repository.

## Overview

`github.com/hallode/golib` — Go library of infra building blocks: storage clients (PostgreSQL via pgx, Redis, MongoDB), observability (OTel tracing, zap logging, Prometheus middleware), error handling, validation, and integrations (Slack, email, Excel). Library only (no `main` package), imported by other services.

## Commands

```bash
go build ./...            # compile everything
go vet ./...              # static checks
go test ./...             # run all tests
go test -race ./...       # tests with race detector
go mod tidy               # sync go.mod/go.sum after adding/removing imports
```

## Module path gotcha

Module path: `github.com/hallode/golib` (git remote: `git@github.com:hallode/golib.git`). Never append `.git` to the module path or import paths.

## Architecture

Flat top-level packages, loosely coupled. Package dependency graph:

```
json ← log ← circuitbreaker
custerr ← log            slack → json, log
redis → json             excel → json (one call site)
validator → custerr
otel, fprom, email, storage/* → no other golib deps
```

### Package reference

- **`custerr`** — Structured application errors: `Wrap`/`Wrapf` (source capture), `Custer` (HTTP-status + optional business code — useful for REST/gRPC gateways), `NonRetryableError` (marks errors that retry loops should skip), `Sourcer`/`Messager`. Generic business codes (format `CCDDNNN`); services attach domain codes via `WithCode()`. `RegisterStatusFallback()` at startup only. `SetAppModule("my-service/")` shortens captured source paths.
- **`json`** — Swappable codec over `encoding/json`: `Init("sonic")` switches to bytedance/sonic globally.
- **`log`** — zap global singleton: `log.New` or `log.NewWithConfig` (optional `EnableTraceID` for OTel). Injects `trace_id` when enabled; `Sanitize()` redacts secrets.
- **`circuitbreaker`** — sony/gobreaker wrapper. `NonTransientError` does not trip the breaker. Optional logging via `EnableLogging`. `Execute[T]()`.
- **`redis`** — go-redis/v9. `NewRedis` / `NewRedisSentinel`. Optional OTel via `EnableOpenTelemetry` (redisotel tracing + metrics). Per-call options in `redis/options` (TTL, jitter, MsgPack/JSON). Hash ops, pipelined ops, pattern delete.
- **`otel`** — Tracer provider: `"otlp"` / `"stdout"` / `"no-op"`. `Tracer(ctx)` uses `runtime.Caller(1)` — call directly, do not wrap. `ContextWithUnitTest(ctx)` for tests.
- **`validator`** — go-playground/validator/v10 plus generic and domain-specific custom rules (EN/ID). `FiberValidator` for Fiber v3. Extend via `custom_validator.go` chain or a custom instance.
- **`slack`** — Webhook client + async `AlertWorker` (aggregation, rate limit, circuit breaker).
- **`excel`** — excelize/v2: export, streaming sheets, multi-sheet workbooks.
- **`fprom`** — Fiber v3 Prometheus middleware.
- **`storage/sql`** — PostgreSQL via **pgx/v5** `pgxpool`. `New(opt) (*SQL, error)`, `Close()`, `Ping(ctx)`. Exposes `Pool *pgxpool.Pool`.
- **`storage/mongo`** — **mongo-driver/v2**. Optional OTel via `EnableOpenTelemetry` (otelmongo). `New(opt) (*Mongo, error)`, `Close(ctx)`, `Ping(ctx)`, `Client()` for transactions. Exposes `DB *mongo.Database`.
- **`email`** — gomail SMTP `Client` + fluent `Message` (text/HTML, attachments, CC/BCC). `SendContext(ctx, msg)`.

## Service startup

Typical initialization when using logging, tracing, and storage together:

1. `log.New("svc")` or `log.NewWithConfig(log.Config{ServiceName: "svc", EnableTraceID: true})`
2. `otel.NewTracer(cfg)` when exporting traces
3. `json.Init("sonic")`, `custerr.SetAppModule(...)`, `custerr.RegisterStatusFallback(...)` as needed
4. `EnableOpenTelemetry: true` on redis/mongo clients when step 2 is configured

Call `mongo.Close(ctx)` and `sql.Close()` on shutdown.

## Optional integrations

| Feature | Config |
|---------|--------|
| Redis OTel | `RedisOption.EnableOpenTelemetry` |
| Mongo OTel | `mongo.Options.EnableOpenTelemetry` |
| Log trace_id | `log.Config.EnableTraceID` |
| Sonic JSON | `json.Init("sonic")` |
| Redis MsgPack | `UseMsgPack` or `options.WithMsgPack()` |
| CB logging | `circuitbreaker.Config.EnableLogging` |

## Testing

Packages with tests: `circuitbreaker`, `custerr`, `email`, `excel`, `json`, `log`, `otel`, `redis`, `redis/options`, `slack`, `storage/mongo`, `storage/sql`, `validator`.

Redis tests use `miniredis`. Mongo/SQL tests cover validation and connection failure (no live DB required in CI).

## Known constraints

- **`otelmongo` v2** — pinned to contrib pseudo-version until a stable tag is published.
- **`vendor/`** — gitignored. On vendoring errors: `rm -rf vendor && go mod tidy`.
- **Scope** — minimal diffs; match existing patterns; no over-engineering.
