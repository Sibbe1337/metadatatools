package middleware

import (
	"github.com/gin-gonic/gin"
	"go.opentelemetry.io/otel"
)

// Tracing returns a middleware that adds OpenTelemetry tracing to requests
func Tracing() gin.HandlerFunc {
	return func(c *gin.Context) {
		tracer := otel.Tracer("http")
		ctx, span := tracer.Start(c.Request.Context(), c.Request.URL.Path)
		defer span.End()

		c.Request = c.Request.WithContext(ctx)
		c.Next()
	}
}
