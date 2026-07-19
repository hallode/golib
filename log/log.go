// Package log is a zap-based structured logger exposed as a global singleton.
// Call New or NewWithConfig once at startup before using the package-level
// helpers (otherwise they no-op). NewWithConfig with EnableTraceID injects the
// OpenTelemetry trace_id (pairs with golib/otel); Sanitize redacts secrets from
// logged values.
package log

import (
	"context"
	"errors"
	"regexp"
	"strings"

	"github.com/hallode/golib/custerr"
	"github.com/hallode/golib/json"

	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type Params map[string]any

type Logger interface {
	With(ctx context.Context) Logger
	WithStack(err error) Logger
	WithParam(key string, value any) Logger
	WithParams(params Params) Logger
	Errorf(format string, args ...any)
	Error(args ...any)
	Fatalf(format string, args ...any)
	Fatal(args ...any)
	Infof(format string, args ...any)
	Info(args ...any)
	Warnf(format string, args ...any)
	Warn(args ...any)
	Debugf(format string, args ...any)
	Debug(args ...any)
}

type logger struct {
	s             *zap.SugaredLogger
	enableTraceID bool
}

var logStore *logger
var atomicLevel zap.AtomicLevel

// sensitivePattern pairs a regex with a fixed replacement name.
// If name is empty, the first capture group from the match is used (uppercased).
type sensitivePattern struct {
	re   *regexp.Regexp
	name string
}

var SensitivePatterns = []sensitivePattern{
	{regexp.MustCompile(`(?i)(password|passwd|pwd)\b[^:=]*[:=]\s*\S+`), ""},
	{regexp.MustCompile(`(?i)(token|bearer|authorization)\b[^:=]*[:=]\s*\S+`), ""},
	{regexp.MustCompile(`(?i)(api_key|apikey|api-key)\b[^:=]*[:=]\s*\S+`), ""},
	{regexp.MustCompile(`(?i)(secret|private_key|private-key)\b[^:=]*[:=]\s*\S+`), ""},
	{regexp.MustCompile(`(?i)(credential|credit_card|credit-card)\b[^:=]*[:=]\s*\S+`), ""},
	{regexp.MustCompile(`Bearer\s+[A-Za-z0-9\-._~+/]+=*`), "BEARER"},
	{regexp.MustCompile(`Basic\s+[A-Za-z0-9+/=]+`), "BASIC"},
}

const sanitizeMaxBytes = 2000

// Config configures the global logger.
type Config struct {
	ServiceName   string
	EnableTraceID bool // set true when using golib/otel and trace_id fields are desired
}

// Sanitize returns a sanitized string representation of v.
// If v is a string or []byte it is used directly; otherwise it is JSON-marshaled.
// Sensitive field values are replaced with *FIELDNAME* (e.g. "password":"x" → *PASSWORD*).
// The result is truncated to 2000 bytes.
// Returns "" if v cannot be marshaled.
func Sanitize(v any) string {
	var s string
	switch val := v.(type) {
	case string:
		s = val
	case []byte:
		s = string(val)
	default:
		b, err := json.Marshal(v)
		if err != nil {
			return ""
		}
		s = string(b)
	}

	for _, p := range SensitivePatterns {
		p := p // capture
		s = p.re.ReplaceAllStringFunc(s, func(match string) string {
			if p.name != "" {
				return "***" + p.name + "***"
			}
			subs := p.re.FindStringSubmatch(match)
			if len(subs) > 1 {
				return "***" + strings.ToUpper(subs[1]) + "***"
			}
			return "***REDACTED***"
		})
	}

	if len(s) > sanitizeMaxBytes {
		s = s[:sanitizeMaxBytes] + "... (truncated)"
	}
	return s
}

func IsDebug() bool {
	if logStore == nil {
		return false
	}
	return atomicLevel.Level() <= zapcore.DebugLevel
}

// New initializes the global logger with JSON (production) encoding at info level.
func New(serviceName string) Logger {
	return NewWithConfig(Config{ServiceName: serviceName})
}

// NewWithConfig initializes the global logger with optional OpenTelemetry trace_id injection.
func NewWithConfig(cfg Config) Logger {
	atomicLevel = zap.NewAtomicLevelAt(zapcore.InfoLevel)

	zapCfg := zap.NewProductionConfig()
	zapCfg.Level = atomicLevel
	zapCfg.EncoderConfig.TimeKey = "time"
	zapCfg.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder

	base, err := zapCfg.Build()
	if err != nil {
		panic("log.New: failed to build zap logger: " + err.Error())
	}

	logStore = &logger{
		s:             base.Sugar().With("service", cfg.ServiceName),
		enableTraceID: cfg.EnableTraceID,
	}
	return logStore
}

// SetLevel changes the global logger's minimum level at runtime.
// Accepted values: debug, info, warn (or warning), error, fatal, panic;
// anything else falls back to info.
func SetLevel(level string) {
	var l zapcore.Level
	switch level {
	case "debug":
		l = zapcore.DebugLevel
	case "info":
		l = zapcore.InfoLevel
	case "warning", "warn":
		l = zapcore.WarnLevel
	case "error":
		l = zapcore.ErrorLevel
	case "fatal":
		l = zapcore.FatalLevel
	case "panic":
		l = zapcore.PanicLevel
	default:
		l = zapcore.InfoLevel
	}
	atomicLevel.SetLevel(l)
}

func (l *logger) With(ctx context.Context) Logger {
	if ctx == nil || !l.enableTraceID {
		return l
	}

	spanCtx := trace.SpanContextFromContext(ctx)
	if !spanCtx.IsValid() {
		return l
	}

	return &logger{
		s:             l.s.With("trace_id", spanCtx.TraceID().String()),
		enableTraceID: l.enableTraceID,
	}
}

// WithStack attaches a "source" field from custerr.Sourcer in the error chain, or from the call site.
func (l *logger) WithStack(err error) Logger {
	if err == nil {
		return l
	}

	var sourcer custerr.Sourcer
	if errors.As(err, &sourcer) {
		if src := sourcer.Source(); src != "" {
			return &logger{s: l.s.With("source", src), enableTraceID: l.enableTraceID}
		}
	}

	if src := custerr.CaptureSource(2); src != "" {
		return &logger{s: l.s.With("source", src), enableTraceID: l.enableTraceID}
	}

	return l
}

func (l *logger) WithParam(key string, value any) Logger {
	return &logger{s: l.s.With(key, value), enableTraceID: l.enableTraceID}
}

func (l *logger) WithParams(params Params) Logger {
	args := make([]any, 0, len(params)*2)
	for k, v := range params {
		args = append(args, k, v)
	}
	return &logger{s: l.s.With(args...), enableTraceID: l.enableTraceID}
}

func (l *logger) Errorf(format string, args ...any) { l.s.Errorf(format, args...) }
func (l *logger) Error(args ...any)                 { l.s.Error(args...) }
func (l *logger) Fatalf(format string, args ...any) { l.s.Fatalf(format, args...) }
func (l *logger) Fatal(args ...any)                 { l.s.Fatal(args...) }
func (l *logger) Infof(format string, args ...any)  { l.s.Infof(format, args...) }
func (l *logger) Info(args ...any)                  { l.s.Info(args...) }
func (l *logger) Warnf(format string, args ...any)  { l.s.Warnf(format, args...) }
func (l *logger) Warn(args ...any)                  { l.s.Warn(args...) }
func (l *logger) Debugf(format string, args ...any) { l.s.Debugf(format, args...) }
func (l *logger) Debug(args ...any)                 { l.s.Debug(args...) }

func GetLogger() Logger {
	if logStore == nil {
		return nil
	}
	return logStore
}

func With(ctx context.Context) Logger {
	if logStore == nil {
		return &logger{s: zap.NewNop().Sugar()}
	}
	return logStore.With(ctx)
}

func WithParams(params Params) Logger {
	if logStore == nil {
		return &logger{s: zap.NewNop().Sugar()}
	}
	return logStore.WithParams(params)
}

func WithStack(err error) Logger {
	if logStore == nil {
		return &logger{s: zap.NewNop().Sugar()}
	}
	return logStore.WithStack(err)
}

func Info(args ...any) {
	if logStore != nil {
		logStore.Info(args...)
	}
}

func Infof(format string, args ...any) {
	if logStore != nil {
		logStore.Infof(format, args...)
	}
}

func Error(args ...any) {
	if logStore != nil {
		logStore.Error(args...)
	}
}

func Errorf(format string, args ...any) {
	if logStore != nil {
		logStore.Errorf(format, args...)
	}
}

func Warn(args ...any) {
	if logStore != nil {
		logStore.Warn(args...)
	}
}

func Warnf(format string, args ...any) {
	if logStore != nil {
		logStore.Warnf(format, args...)
	}
}

func Debug(args ...any) {
	if logStore != nil {
		logStore.Debug(args...)
	}
}

func Debugf(format string, args ...any) {
	if logStore != nil {
		logStore.Debugf(format, args...)
	}
}

func Fatal(args ...any) {
	if logStore != nil {
		logStore.Fatal(args...)
	}
}

func Fatalf(format string, args ...any) {
	if logStore != nil {
		logStore.Fatalf(format, args...)
	}
}
