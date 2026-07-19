package slack

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/hallode/golib/v2/log"

	"github.com/pkg/errors"
)

const (
	maxAlertsPerMinute   = 10
	aggregationWindow    = 1 * time.Minute
	maxRetries           = 3
	baseRetryDelay       = 500 * time.Millisecond
	maxRequestBodyLength = 1000
	maxStackTraceLines   = 30
	maxErrorLength       = 500
)

type AlertSeverity string

const (
	SeverityPanic   AlertSeverity = "panic"
	SeverityFatal   AlertSeverity = "fatal"
	SeverityError   AlertSeverity = "error"
	SeverityWarning AlertSeverity = "warning"
)

type Alert struct {
	Severity      AlertSeverity
	AppName       string
	Source        string
	Message       string
	Method        string
	Path          string
	QueryParams   string
	StatusCode    int
	TraceID       string
	StackTrace    string
	RequestBody   string
	CallerService string
	ErrorSource   string
	Timestamp     time.Time
}

type AlertWorker struct {
	client         *Client
	alertChan      chan Alert
	wg             sync.WaitGroup
	stopChan       chan struct{}
	rateLimiter    *rateLimiter
	aggregator     *alertAggregator
	circuitBreaker *circuitBreaker
	logger         logger
}

type logger interface {
	Errorf(format string, args ...any)
	Infof(format string, args ...any)
	Warnf(format string, args ...any)
}

func NewAlertWorker(client *Client, log logger) *AlertWorker {
	return &AlertWorker{
		client:         client,
		alertChan:      make(chan Alert, 100),
		stopChan:       make(chan struct{}),
		rateLimiter:    newRateLimiter(maxAlertsPerMinute, time.Minute),
		aggregator:     newAlertAggregator(aggregationWindow),
		circuitBreaker: newCircuitBreaker(5, time.Minute),
		logger:         log,
	}
}

func (w *AlertWorker) Start() {
	w.wg.Add(1)
	go func() {
		defer w.wg.Done()
		ticker := time.NewTicker(100 * time.Millisecond)
		defer ticker.Stop()

		for {
			select {
			case <-w.stopChan:
				return
			case alert := <-w.alertChan:
				w.processAlert(alert)
			case <-ticker.C:
				w.aggregator.flush(w)
			}
		}
	}()
}

func (w *AlertWorker) Stop() {
	close(w.stopChan)
	w.wg.Wait()
	for {
		select {
		case alert := <-w.alertChan:
			_ = w.sendAlert(alert)
		default:
			close(w.alertChan)
			return
		}
	}
}

func (w *AlertWorker) SendAlert(ctx context.Context, alert Alert) {
	alert.Timestamp = time.Now()

	select {
	case w.alertChan <- alert:
	default:
		w.logger.Warnf("Alert channel full, dropping alert: %s", alert.Message)
	}
}

func (w *AlertWorker) processAlert(alert Alert) {
	if w.circuitBreaker.isOpen() {
		w.logger.Warnf("Circuit breaker open, skipping alert: %s", alert.Message)
		return
	}

	if !w.rateLimiter.allow() {
		w.aggregator.add(alert)
		return
	}

	w.aggregator.flush(w)

	if err := w.sendAlert(alert); err != nil {
		w.logger.Errorf("Failed to send alert: %v", err)
		w.circuitBreaker.recordFailure()
	} else {
		w.circuitBreaker.recordSuccess()
	}
}

func (w *AlertWorker) sendAlert(alert Alert) error {
	var message string
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	switch alert.Severity {
	case SeverityPanic:
		message = w.formatPanicAlert(alert)
	case SeverityFatal:
		message = w.formatFatalAlert(alert)
	default:
		message = w.formatGenericAlert(alert)
	}

	return w.retrySend(ctx, message)
}

func (w *AlertWorker) retrySend(ctx context.Context, message string) error {
	var lastErr error
	for attempt := 0; attempt < maxRetries; attempt++ {
		if attempt > 0 {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(backoffDelay(attempt)):
			}
		}

		err := w.client.Send(ctx, message)
		if err == nil {
			return nil
		}
		lastErr = err
	}
	return lastErr
}

func backoffDelay(attempt int) time.Duration {
	delay := baseRetryDelay * time.Duration(1<<uint(attempt))
	if delay > 5*time.Second {
		return 5 * time.Second
	}
	return delay
}

func (w *AlertWorker) formatPanicAlert(alert Alert) string {
	timestamp := alert.Timestamp.In(getJakartaLocation()).Format("2006-01-02 15:04:05 WIB")

	sb := strings.Builder{}
	sb.WriteString(fmt.Sprintf("🚨 *PANIC DETECTED - %s*\n", alert.AppName))
	sb.WriteString(fmt.Sprintf("_%s_\n\n", timestamp))
	if alert.Source != "" {
		sb.WriteString(fmt.Sprintf("*Source:* `%s`\n", alert.Source))
	}
	sb.WriteString(fmt.Sprintf("*Error:* `%s`\n", sanitizeErrorMessage(alert.Message, maxErrorLength)))
	if alert.ErrorSource != "" {
		sb.WriteString(fmt.Sprintf("*Error Location:* `%s`\n", alert.ErrorSource))
	}
	if alert.Method != "" {
		sb.WriteString(fmt.Sprintf("*Method:* `%s`\n", alert.Method))
	}
	if alert.Path != "" {
		sb.WriteString(fmt.Sprintf("*Path:* `%s`\n", alert.Path))
	}
	if alert.StatusCode > 0 {
		sb.WriteString(fmt.Sprintf("*Status Code:* `%d`\n", alert.StatusCode))
	}
	if alert.TraceID != "" {
		sb.WriteString(fmt.Sprintf("*Trace ID:* `%s`\n", alert.TraceID))
	}
	if alert.CallerService != "" {
		sb.WriteString(fmt.Sprintf("*Caller:* `%s`", alert.CallerService))
	}

	if sanitized := sanitizeQueryParams(alert.QueryParams); hasMeaningfulContent(sanitized) {
		sb.WriteString(fmt.Sprintf("\n*Query Params:* `%s`", sanitized))
	}

	if sanitized := sanitizeRequestBody(alert.RequestBody); hasMeaningfulContent(sanitized) {
		sb.WriteString(fmt.Sprintf("\n\n*Request Body:*\n```%s```", truncateString(sanitized, maxRequestBodyLength)))
	}

	if alert.StackTrace != "" {
		truncated := truncateStackTrace(alert.StackTrace, maxStackTraceLines)
		sb.WriteString(fmt.Sprintf("\n\n*Stack Trace:*\n```%s```", truncated))
	}

	return sb.String()
}

func (w *AlertWorker) formatFatalAlert(alert Alert) string {
	timestamp := alert.Timestamp.In(getJakartaLocation()).Format("2006-01-02 15:04:05 WIB")

	sb := strings.Builder{}
	sb.WriteString(fmt.Sprintf("🔴 *FATAL ERROR - %s*\n", alert.AppName))
	sb.WriteString(fmt.Sprintf("_%s_\n\n", timestamp))
	if alert.Source != "" {
		sb.WriteString(fmt.Sprintf("*Source:* `%s`\n", alert.Source))
	}
	sb.WriteString(fmt.Sprintf("*Error:* `%s`\n", sanitizeErrorMessage(alert.Message, maxErrorLength)))
	if alert.ErrorSource != "" {
		sb.WriteString(fmt.Sprintf("*Error Location:* `%s`\n", alert.ErrorSource))
	}
	if alert.Method != "" {
		sb.WriteString(fmt.Sprintf("*Method:* `%s`\n", alert.Method))
	}
	if alert.Path != "" {
		sb.WriteString(fmt.Sprintf("*Path:* `%s`\n", alert.Path))
	}
	if alert.StatusCode > 0 {
		sb.WriteString(fmt.Sprintf("*Status Code:* `%d`\n", alert.StatusCode))
	}
	if alert.TraceID != "" {
		sb.WriteString(fmt.Sprintf("*Trace ID:* `%s`\n", alert.TraceID))
	}
	if alert.CallerService != "" {
		sb.WriteString(fmt.Sprintf("*Caller:* `%s`", alert.CallerService))
	}

	if sanitized := sanitizeQueryParams(alert.QueryParams); hasMeaningfulContent(sanitized) {
		sb.WriteString(fmt.Sprintf("\n*Query Params:* `%s`", sanitized))
	}

	if sanitized := sanitizeRequestBody(alert.RequestBody); hasMeaningfulContent(sanitized) {
		sb.WriteString(fmt.Sprintf("\n\n*Request Body:*\n```%s```", truncateString(sanitized, maxRequestBodyLength)))
	}

	if alert.StackTrace != "" {
		truncated := truncateStackTrace(alert.StackTrace, maxStackTraceLines)
		sb.WriteString(fmt.Sprintf("\n\n*Stack Trace:*\n```%s```", truncated))
	}

	return sb.String()
}

func (w *AlertWorker) formatGenericAlert(alert Alert) string {
	timestamp := alert.Timestamp.In(getJakartaLocation()).Format("2006-01-02 15:04:05 WIB")

	sb := strings.Builder{}
	emoji := getEmojiForSeverity(alert.Severity)
	sb.WriteString(fmt.Sprintf("%s *%s - %s*\n", emoji, strings.ToUpper(string(alert.Severity)), alert.AppName))
	sb.WriteString(fmt.Sprintf("_%s_\n\n", timestamp))
	if alert.Source != "" {
		sb.WriteString(fmt.Sprintf("*Source:* `%s`\n", alert.Source))
	}
	sb.WriteString(fmt.Sprintf("*Error:* `%s`\n", sanitizeErrorMessage(alert.Message, maxErrorLength)))
	if alert.Method != "" {
		sb.WriteString(fmt.Sprintf("*Method:* `%s`\n", alert.Method))
	}
	if alert.Path != "" {
		sb.WriteString(fmt.Sprintf("*Path:* `%s`\n", alert.Path))
	}
	if alert.StatusCode > 0 {
		sb.WriteString(fmt.Sprintf("*Status Code:* `%d`\n", alert.StatusCode))
	}
	if alert.TraceID != "" {
		sb.WriteString(fmt.Sprintf("*Trace ID:* `%s`\n", alert.TraceID))
	}
	if alert.CallerService != "" {
		sb.WriteString(fmt.Sprintf("*Caller:* `%s`", alert.CallerService))
	}

	if sanitized := sanitizeQueryParams(alert.QueryParams); hasMeaningfulContent(sanitized) {
		sb.WriteString(fmt.Sprintf("\n*Query Params:* `%s`", sanitized))
	}

	if sanitized := sanitizeRequestBody(alert.RequestBody); hasMeaningfulContent(sanitized) {
		sb.WriteString(fmt.Sprintf("\n\n*Request Body:*\n```%s```", truncateString(sanitized, maxRequestBodyLength)))
	}

	return sb.String()
}

func sanitizeRequestBody(body string) string {
	return log.Sanitize(body)
}

func sanitizeQueryParams(params string) string {
	return log.Sanitize(params)
}

// hasMeaningfulContent returns false for blank strings and empty JSON
// structures (e.g. "{}", "[]", "null") that carry no useful information.
func hasMeaningfulContent(s string) bool {
	s = strings.TrimSpace(s)
	switch s {
	case "", "{}", "[]", "null":
		return false
	}
	return true
}

func sanitizeErrorMessage(msg string, maxLen int) string {
	msg = strings.TrimSpace(msg)
	if msg == "" {
		return "Unknown error"
	}
	if len(msg) > maxLen {
		msg = msg[:maxLen] + "..."
	}
	return msg
}

func getEmojiForSeverity(severity AlertSeverity) string {
	switch severity {
	case SeverityPanic:
		return "🚨"
	case SeverityFatal:
		return "🔴"
	case SeverityError:
		return "⚠️"
	case SeverityWarning:
		return "⚡"
	default:
		return "ℹ️"
	}
}

func getJakartaLocation() *time.Location {
	loc, err := time.LoadLocation("Asia/Jakarta")
	if err != nil {
		return time.FixedZone("WIB", 7*60*60)
	}
	return loc
}

func truncateStackTrace(stackTrace string, maxLines int) string {
	lines := strings.Split(stackTrace, "\n")
	if len(lines) <= maxLines {
		return stackTrace
	}
	firstLines := lines[:maxLines]
	truncated := strings.Join(firstLines, "\n")
	truncated += fmt.Sprintf("\n... (truncated, total %d lines)", len(lines))
	return truncated
}

func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "... (truncated)"
}

func ExtractErrorMessage(err error) string {
	if err == nil {
		return ""
	}

	errMsg := err.Error()
	if errMsg == "" {
		return "An unexpected error occurred"
	}

	lowerErrMsg := strings.ToLower(errMsg)
	if lowerErrMsg == "internal server error" || lowerErrMsg == "unknown error" {
		return "An unexpected error occurred"
	}

	return sanitizeErrorMessage(errMsg, maxErrorLength)
}

// ExtractErrorSource walks the error chain and returns the source location
// (file:line func) from the outermost error that implements custerr.Sourcer.
func ExtractErrorSource(err error) string {
	if err == nil {
		return ""
	}

	type sourcer interface {
		Source() string
	}
	type unwrapper interface {
		Unwrap() error
	}

	for err != nil {
		if s, ok := err.(sourcer); ok {
			if src := s.Source(); src != "" {
				return src
			}
		}
		u, ok := err.(unwrapper)
		if !ok {
			break
		}
		err = u.Unwrap()
	}

	return ""
}

var (
	globalWorker     *AlertWorker
	globalWorkerOnce sync.Once
	globalWorkerMu   sync.RWMutex
)

func InitAlertWorker(client *Client, log logger) {
	globalWorkerOnce.Do(func() {
		w := NewAlertWorker(client, log)
		w.Start()
		globalWorkerMu.Lock()
		globalWorker = w
		globalWorkerMu.Unlock()
	})
}

func StopAlertWorker() {
	globalWorkerMu.RLock()
	w := globalWorker
	globalWorkerMu.RUnlock()
	if w != nil {
		w.Stop()
	}
}

func SendAlert(ctx context.Context, alert Alert) error {
	globalWorkerMu.RLock()
	w := globalWorker
	globalWorkerMu.RUnlock()
	if w == nil {
		return errors.New("alert worker not initialized")
	}
	w.SendAlert(ctx, alert)
	return nil
}

type AlertBuilder struct {
	alert Alert
}

func BuildAlert(severity AlertSeverity, appName, message string) *AlertBuilder {
	return &AlertBuilder{
		alert: Alert{
			Severity:  severity,
			AppName:   appName,
			Message:   message,
			Timestamp: time.Now(),
		},
	}
}

func (b *AlertBuilder) WithMethod(method string) *AlertBuilder {
	b.alert.Method = method
	return b
}

func (b *AlertBuilder) WithPath(path string) *AlertBuilder {
	b.alert.Path = path
	return b
}

func (b *AlertBuilder) WithQueryParams(params string) *AlertBuilder {
	b.alert.QueryParams = params
	return b
}

func (b *AlertBuilder) WithStatusCode(code int) *AlertBuilder {
	b.alert.StatusCode = code
	return b
}

func (b *AlertBuilder) WithTraceID(traceID string) *AlertBuilder {
	b.alert.TraceID = traceID
	return b
}

func (b *AlertBuilder) WithStackTrace(stack string) *AlertBuilder {
	b.alert.StackTrace = stack
	return b
}

func (b *AlertBuilder) WithRequestBody(body string) *AlertBuilder {
	b.alert.RequestBody = body
	return b
}

func (b *AlertBuilder) WithCallerService(caller string) *AlertBuilder {
	b.alert.CallerService = caller
	return b
}

func (b *AlertBuilder) WithSource(source string) *AlertBuilder {
	b.alert.Source = source
	return b
}

func (b *AlertBuilder) WithErrorSource(source string) *AlertBuilder {
	b.alert.ErrorSource = source
	return b
}

func (b *AlertBuilder) Send(ctx context.Context) error {
	return SendAlert(ctx, b.alert)
}

func (b *AlertBuilder) Build() Alert {
	return b.alert
}
