// Package redis is an opinionated cache wrapper over go-redis/v9 with pluggable
// value serialization (JSON by default, MessagePack opt-in) and TTL jitter.
//
// Construct a client with NewRedis (standalone) or NewRedisSentinel (failover).
// The returned *Redis embeds *redis.Client, so every raw go-redis method is also
// available. Typed helpers (Get, Set, MGet, hash and pipelined ops) serialize
// values for you; per-call options.Option overrides let you tune TTL/format.
//
// Two things to keep consistent: the serialization format must match on write
// and read (a value written as MessagePack cannot be read as JSON), and
// RedisOption.Exp is expressed in seconds, not a time.Duration. EnableOpenTelemetry
// requires a globally configured OTel tracer provider (see golib/otel).
package redis
