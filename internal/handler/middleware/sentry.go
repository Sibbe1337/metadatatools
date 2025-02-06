package middleware

import (
	"fmt"
	"metadatatool/internal/pkg/domain"
	"metadatatool/internal/pkg/errortracking"
	"runtime/debug"
	"time"

	"github.com/getsentry/sentry-go"
	"github.com/gofiber/fiber/v2"
)

// SentryMiddleware creates a middleware for Sentry error tracking
func SentryMiddleware() fiber.Handler {
	errorTracker := errortracking.NewErrorTracker()

	return func(c *fiber.Ctx) error {
		// Start a new transaction
		hub := sentry.CurrentHub().Clone()
		transaction := sentry.StartTransaction(
			c.Context(),
			fmt.Sprintf("%s %s", c.Method(), c.Route().Path),
		)
		defer transaction.Finish()

		// Set transaction on hub
		hub.ConfigureScope(func(scope *sentry.Scope) {
			scope.SetTag("http.method", c.Method())
			scope.SetTag("http.url", c.OriginalURL())
			scope.SetTag("http.route", c.Route().Path)
			scope.SetTag("http.host", c.Hostname())
			scope.SetTag("http.remote_addr", c.IP())

			// Add user context if available
			if user := c.Locals("user"); user != nil {
				if claims, ok := user.(*domain.Claims); ok {
					scope.SetUser(sentry.User{
						ID:       claims.UserID,
						Email:    claims.Email,
						Username: claims.Email,
					})
				}
			}

			// Add request data as tags since we can't use SetRequest directly
			scope.SetTag("request.host", string(c.Request().Host()))
			scope.SetTag("request.uri", string(c.Request().RequestURI()))
		})

		// Create a child span for the handler execution
		span := transaction.StartChild("handler.execution")
		start := time.Now()

		// Handle panics
		defer func() {
			if err := recover(); err != nil {
				stack := debug.Stack()

				hub.ConfigureScope(func(scope *sentry.Scope) {
					scope.SetLevel(sentry.LevelFatal)
					scope.SetExtra("stacktrace", string(stack))
				})

				hub.RecoverWithContext(
					c.Context(),
					err,
				)

				// Ensure event is sent before panic continues
				hub.Flush(2 * time.Second)

				// Re-panic after sending to Sentry
				panic(err)
			}
		}()

		// Process request
		err := c.Next()

		// Finish span
		span.Finish()

		// Record request duration as a tag since SetMeasurement is not available
		duration := time.Since(start)
		hub.ConfigureScope(func(scope *sentry.Scope) {
			scope.SetTag("request.duration_ms", fmt.Sprintf("%d", duration.Milliseconds()))
		})

		// Capture any errors
		if err != nil {
			hub.ConfigureScope(func(scope *sentry.Scope) {
				scope.SetLevel(sentry.LevelError)
				scope.SetTag("http.status_code", fmt.Sprintf("%d", c.Response().StatusCode()))
				scope.SetExtra("request.params", c.AllParams())
				scope.SetExtra("request.query", c.Queries())
			})

			errorTracker.CaptureError(err, map[string]string{
				"component": "http",
				"method":    c.Method(),
				"path":      c.Route().Path,
			})
		}

		return err
	}
}
