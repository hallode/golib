# golib

A Go library of infrastructure building blocks — storage, observability, structured errors, validation, and service integrations. Designed as a reusable module for microservices and backend applications.

```bash
go get github.com/hallode/golib/v2
```

**Requirements:** Go 1.26+

> **Free & open-source** — golib is released under the [MIT](LICENSE) license.
> Use it in anything, including commercial and closed-source projects, no strings
> attached. If it helps your work, please ⭐ [star the repo](https://github.com/hallode/golib)
> so others can find it.

## Packages

| Package | Import path | Description |
|---------|-------------|-------------|
| Errors | `github.com/hallode/golib/v2/custerr` | Wrap with source, HTTP-status errors, non-retryable marker, business codes |
| JSON | `github.com/hallode/golib/v2/json` | Pluggable JSON codec (stdlib or Sonic) |
| Log | `github.com/hallode/golib/v2/log` | Structured zap logger; optional trace ID; secret redaction via `Sanitize` |
| Circuit breaker | `github.com/hallode/golib/v2/circuitbreaker` | gobreaker wrapper with non-transient errors |
| Redis | `github.com/hallode/golib/v2/redis` | go-redis v9; optional OTel via `EnableOpenTelemetry`; options in `redis/options` |
| OTel | `github.com/hallode/golib/v2/otel` | Tracer provider (OTLP, stdout, no-op) |
| Validator | `github.com/hallode/golib/v2/validator` | validator/v10 + custom rules (EN/ID), Fiber hook |
| Slack | `github.com/hallode/golib/v2/slack` | Webhook client and async alert worker |
| Excel | `github.com/hallode/golib/v2/excel` | excelize streaming export helpers |
| Prometheus | `github.com/hallode/golib/v2/fprom` | Fiber v3 HTTP metrics middleware |
| Email | `github.com/hallode/golib/v2/email` | SMTP client (text/HTML, attachments) |
| Cursor | `github.com/hallode/golib/v2/cursor` | Storage-agnostic cursor (keyset) pagination |
| PostgreSQL | `github.com/hallode/golib/v2/storage/sql` | pgx/v5 connection pool |
| MongoDB | `github.com/hallode/golib/v2/storage/mongo` | mongo-driver v2; optional OTel via `EnableOpenTelemetry` |

## Quick start

### Service bootstrap

```go
import (
    "github.com/hallode/golib/v2/custerr"
    "github.com/hallode/golib/v2/json"
    "github.com/hallode/golib/v2/log"
    "github.com/hallode/golib/v2/otel"
)

func main() {
    log.NewWithConfig(log.Config{
        ServiceName:   "my-service",
        EnableTraceID: true,
    })
    otel.NewTracer(&otel.TracerConfig{
        Name:         "my-service",
        Tracer:       "otlp",
        OtelEndpoint: "http://localhost:4318", // OTLP HTTP, not gRPC :4317
    })
    json.Init("sonic")
    custerr.SetAppModule("my-service/")
}
```

Minimal bootstrap (logging only):

```go
log.New("my-service")
```

See [Optional integrations](#optional-integrations) for flags that default to off.

### PostgreSQL (pgx)

```go
import "github.com/hallode/golib/v2/storage/sql"

db, err := sql.New(&sql.Options{
    DSN: os.Getenv("DATABASE_URL"),
})
if err != nil {
    log.Fatal(err)
}
defer db.Close()

var id int64
err = db.Pool.QueryRow(ctx, "SELECT id FROM users WHERE email = $1", email).Scan(&id)
```

### MongoDB

```go
import "github.com/hallode/golib/v2/storage/mongo"

m, err := mongo.New(&mongo.Options{
    URI:                 os.Getenv("MONGO_URI"),
    DB:                  "app",
    AppName:             "my-service",
    EnableOpenTelemetry: true,
})
if err != nil {
    return err
}
defer m.Close(context.Background())

coll := m.DB.Collection("orders")
```

### Redis

```go
import (
    "github.com/hallode/golib/v2/redis"
    "github.com/hallode/golib/v2/redis/options"
)

r, err := redis.NewRedis(redis.RedisOption{
    Host:                "localhost:6379",
    UseMsgPack:          true,
    EnableOpenTelemetry: true,
})
// ...

var result MyStruct
err = r.Get(ctx, "key", &result, options.WithMsgPack())
```

### Email

```go
import "github.com/hallode/golib/v2/email"

client, err := email.New(email.Config{
    Host:     "smtp.example.com",
    Port:     587,
    Username: "app@example.com",
    Password: os.Getenv("SMTP_PASSWORD"),
    FromName: "My App",
})

err = client.SendContext(ctx,
    client.NewMessage().
        To("user@example.com").
        Subject("Hello").
        Text("Plain body").
        HTML("<p>HTML body</p>"),
)
```

### HTTP errors

```go
import "github.com/hallode/golib/v2/custerr"

return custerr.NotFoundf("order %s not found", id)
return custerr.BadRequest(err).WithCode(myBusinessCode)
```

### Cursor pagination

Storage-agnostic keyset pagination. Your row type implements `Cursor()`, you
over-fetch `Limit+1` rows, and `GetPagination` trims the page and builds the
opaque next/prev cursors. See the [package example](https://pkg.go.dev/github.com/hallode/golib/v2/cursor#example-GetPagination)
for a full backend (including the ORDER-BY flip on backward scans).

```go
import "github.com/hallode/golib/v2/cursor"

type User struct{ ID int64 }
func (u User) Cursor() any { return u.ID }

// rows was fetched with Limit+1 in the scan direction decoded from the request.
page := cursor.GetPagination(cursor.PaginationRequest[User]{
    Cursor: req.Cursor,
    Limit:  req.Limit,
    Next:   req.Next,
    Data:   rows,
})
// page.Data, page.Next, page.Prev
```

## Development

```bash
go build ./...
go vet ./...
go test ./...
go test -race ./...
go mod tidy
```

### Test coverage

Every package has unit tests, and each ships runnable [Go examples](https://go.dev/blog/examples) visible on pkg.go.dev. Redis tests use [miniredis](https://github.com/alicebob/miniredis); mongo and SQL tests cover config validation and connection errors without requiring live databases.

## Optional integrations

Some features are disabled by default and enabled through config:

| Feature | Package | Config |
|---------|---------|--------|
| OpenTelemetry | `redis` | `RedisOption.EnableOpenTelemetry` |
| OpenTelemetry | `storage/mongo` | `Options.EnableOpenTelemetry` |
| Trace ID in logs | `log` | `Config.EnableTraceID` |
| Sonic JSON | `json` | `Init("sonic")` — default is `encoding/json` |
| MsgPack serialization | `redis` | `UseMsgPack` or `options.WithMsgPack()` |
| State-change logging | `circuitbreaker` | `Config.EnableLogging` |
| Prometheus middleware | `fprom` | import when using Fiber |
| Auto validation | `validator` | `FiberValidator` when using Fiber |
| Async alerts | `slack` | `NewAlertWorker` / `InitAlertWorker` |

## Design notes

- **Flat packages** — each directory is a separate import path; no shared `pkg/` tree.
- **Explicit lifecycle** — `storage/sql` and `storage/mongo` return errors from `New`, expose `Close`, and support `Ping`.
- **Composable** — import only the packages you need; enable cross-cutting features through the config above.

## Agent / contributor docs

- [`AGENTS.md`](AGENTS.md) — instructions for AI coding agents
- [`CLAUDE.md`](CLAUDE.md) — Claude Code project context

## License

[MIT](LICENSE) — free for personal and commercial use.
