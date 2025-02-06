package errortracking

import (
	"fmt"
	"os"

	"github.com/getsentry/sentry-go"
)

// ErrorTracker handles error reporting to Sentry
type ErrorTracker struct {
	initialized bool
}

// NewErrorTracker creates a new error tracker
func NewErrorTracker() *ErrorTracker {
	dsn := os.Getenv("SENTRY_DSN")
	if dsn == "" {
		return &ErrorTracker{initialized: false}
	}

	err := sentry.Init(sentry.ClientOptions{
		Dsn:              dsn,
		Environment:      os.Getenv("APP_ENV"),
		TracesSampleRate: 1.0,
	})

	if err != nil {
		fmt.Printf("Sentry initialization failed: %v\n", err)
		return &ErrorTracker{initialized: false}
	}

	return &ErrorTracker{initialized: true}
}

// CaptureError reports an error to Sentry
func (t *ErrorTracker) CaptureError(err error, tags map[string]string) {
	if !t.initialized || err == nil {
		return
	}

	sentry.WithScope(func(scope *sentry.Scope) {
		for key, value := range tags {
			scope.SetTag(key, value)
		}
		sentry.CaptureException(err)
	})
}

// Close flushes any pending events
func (t *ErrorTracker) Close() {
	if t.initialized {
		sentry.Flush(2)
	}
}
