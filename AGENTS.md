# AGENTS.md

Instructions for AI coding agents (Cursor, Claude Code, etc.) working in **golib**.

## Project

| | |
|---|---|
| Module | `github.com/hallode/golib/v2` |
| Type | Go library (no `main` package) |
| Go | 1.26+ |
| Remote | `git@github.com:hallode/golib.git` |

**Never** append `.git` to the module path or import paths.

## Before exploring code

If your setup provides a code-graph / code-indexing MCP tool, prefer it over blind
Grep/Glob for tracing callers, dependents, and test coverage; otherwise use Grep/Read.

## Commands (run before claiming done)

```bash
go build ./...
go vet ./...
go test ./...
```

Use `go test -race ./...` after concurrency-sensitive changes.

## Conventions

1. **Minimal scope** — smallest correct diff; do not refactor unrelated code.
2. **Match existing style** — naming, error wrapping (`fmt.Errorf("pkg: action: %w", err)`), package layout.
3. **Production lifecycle** — storage clients return errors from `New`, support `Close`/`Ping`; no resource leaks.
4. **Comments** — only for non-obvious logic; do not restate function names.
5. **Tests** — add tests for real behavior; skip trivial assertions.
6. **No commits** unless the user explicitly asks.

## Package map

```
custerr/          Structured errors: wrap, HTTP mapping, retry control, business codes
json/             JSON codec (stdlib | sonic)
log/              zap logger; optional OTel trace_id; Sanitize
circuitbreaker/   gobreaker; optional log via EnableLogging
redis/            go-redis wrapper; optional OTel; redis/options
otel/             OTel tracer provider
validator/        go-playground + custom rules
slack/            webhook + alert worker
excel/            excelize helpers
fprom/            Fiber Prometheus middleware
email/            SMTP client (gomail)
storage/sql/      PostgreSQL pgxpool
storage/mongo/    MongoDB driver v2; optional otelmongo
```

## Dependency rules

```
json ← log ← circuitbreaker
custerr ← log            slack → json, log
redis → json             excel → json
validator → custerr
```

Do not introduce circular imports. `storage/*`, `otel`, `fprom`, `email` must not import other golib packages.

## Init order

When multiple packages are used together:

1. `log.New` or `log.NewWithConfig` (`EnableTraceID: true` pairs with `otel`)
2. `otel.NewTracer(cfg)` when exporting traces
3. `json.Init("sonic")`, `custerr.SetAppModule(...)` as needed
4. `EnableOpenTelemetry: true` on redis/mongo when OTel is active
5. `EnableLogging: true` on circuit breaker when log is initialized

## Common pitfalls

- **`otel.Tracer(ctx)`** — call directly in the instrumented function; wrapping breaks span naming (`runtime.Caller`).
- **`log.New`** — initialize before `circuitbreaker` with `EnableLogging: true` or before using slack alert sanitization.
- **`custerr.RegisterStatusFallback`** — call at startup only; not safe for concurrent registration.
- **Mongo `Close(ctx)`** — defer on shutdown; use `Client()` for transactions.
- **SQL `Close()`** — closes the pgx pool; query via `Pool`.

## When adding a new package

- Flat top-level directory under module root.
- No `main` package — library only.
- Update `README.md` and this file if the package is user-facing.
- Add `_test.go` for non-trivial logic.
- Add a `// Package …` doc comment and an `example_test.go` — both render on pkg.go.dev.

## Known constraints

- **`otelmongo` v2** — pinned to a contrib pseudo-version until a stable tag is published.
- **`vendor/`** — gitignored. On vendoring errors: `rm -rf vendor && go mod tidy`.
