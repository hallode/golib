package slack

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"
)

type alertAggregator struct {
	mu            sync.RWMutex
	alerts        map[string]*aggregatedAlert
	flushInterval time.Duration
}

type aggregatedAlert struct {
	count         int
	firstSeen     time.Time
	lastSeen      time.Time
	severity      AlertSeverity
	appName       string
	method        string
	path          string
	message       string
	statusCode    int
	callerService string
	traceID       string
	errorSource   string
	source        string
	queryParams   string
	stackTrace    string
}

func newAlertAggregator(flushInterval time.Duration) *alertAggregator {
	return &alertAggregator{
		alerts:        make(map[string]*aggregatedAlert),
		flushInterval: flushInterval,
	}
}

func (aa *alertAggregator) add(alert Alert) {
	aa.mu.Lock()
	defer aa.mu.Unlock()

	key := aa.generateKey(alert)

	if existing, ok := aa.alerts[key]; ok {
		existing.count++
		existing.lastSeen = alert.Timestamp
	} else {
		aa.alerts[key] = &aggregatedAlert{
			count:         1,
			firstSeen:     alert.Timestamp,
			lastSeen:      alert.Timestamp,
			severity:      alert.Severity,
			appName:       alert.AppName,
			method:        alert.Method,
			path:          alert.Path,
			message:       alert.Message,
			statusCode:    alert.StatusCode,
			callerService: alert.CallerService,
			traceID:       alert.TraceID,
			errorSource:   alert.ErrorSource,
			source:        alert.Source,
			queryParams:   alert.QueryParams,
			stackTrace:    alert.StackTrace,
		}
	}
}

func (aa *alertAggregator) flush(worker *AlertWorker) {
	aa.mu.Lock()
	defer aa.mu.Unlock()

	now := time.Now()
	for key, agg := range aa.alerts {
		if now.Sub(agg.lastSeen) >= aa.flushInterval {
			aa.sendAggregatedAlert(worker, agg)
			delete(aa.alerts, key)
		}
	}
}

func (aa *alertAggregator) generateKey(alert Alert) string {
	msg := alert.Message
	if len(msg) > 100 {
		msg = msg[:100]
	}
	return fmt.Sprintf("%s:%s:%s:%d:%s", alert.Severity, alert.Method, alert.Path, alert.StatusCode, msg)
}

func (aa *alertAggregator) sendAggregatedAlert(worker *AlertWorker, agg *aggregatedAlert) {
	if agg.count == 1 {
		worker.SendAlert(context.Background(), Alert{
			Severity:      agg.severity,
			AppName:       agg.appName,
			Message:       agg.message,
			Method:        agg.method,
			Path:          agg.path,
			StatusCode:    agg.statusCode,
			CallerService: agg.callerService,
			TraceID:       agg.traceID,
			ErrorSource:   agg.errorSource,
			Source:        agg.source,
			QueryParams:   agg.queryParams,
			StackTrace:    agg.stackTrace,
		})
		return
	}

	aggMessage := aa.formatAggregatedMessage(agg)
	worker.SendAlert(context.Background(), Alert{
		Severity:      agg.severity,
		AppName:       agg.appName,
		Message:       aggMessage,
		Method:        agg.method,
		Path:          agg.path,
		StatusCode:    agg.statusCode,
		CallerService: agg.callerService,
		TraceID:       agg.traceID,
		ErrorSource:   agg.errorSource,
		Source:        agg.source,
		QueryParams:   agg.queryParams,
	})
}

func (aa *alertAggregator) formatAggregatedMessage(agg *aggregatedAlert) string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("*Aggregated Alert: %d similar errors*\n\n", agg.count))
	sb.WriteString(fmt.Sprintf("*Original Error:* `%s`\n", sanitizeErrorMessage(agg.message, 300)))
	sb.WriteString(fmt.Sprintf("*First Seen:* %s\n", agg.firstSeen.Format("15:04:05")))
	sb.WriteString(fmt.Sprintf("*Last Seen:* %s\n", agg.lastSeen.Format("15:04:05")))
	return sb.String()
}
