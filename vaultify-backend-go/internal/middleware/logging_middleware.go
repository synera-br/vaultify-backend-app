package middleware

import (
	"net/http" // Added for http status constants
	"time"
	// "log" // Using zap logger instead

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// RequestLogger returns a gin.HandlerFunc (middleware) that logs requests using zap.
// It logs the incoming request method, path, status code, latency, client IP,
// query parameters, and any errors that occurred during the request processing.
func RequestLogger(logger *zap.Logger) gin.HandlerFunc {
	if logger == nil {
		// Fallback to a no-op logger or panic if a logger is absolutely required.
		// For this example, let's assume a logger must be provided.
		panic("RequestLogger requires a non-nil zap.Logger instance")
	}
	return func(c *gin.Context) {
		start := time.Now() // Record start time of the request

		// Make a copy of the path and query to ensure they are not modified by subsequent handlers.
		path := c.Request.URL.Path
		query := c.Request.URL.RawQuery

		// Process request by calling the next handler in the chain.
		// This is crucial to do *before* logging response details like status code and latency.
		c.Next()

		// Log details after the request has been processed by all handlers.
		latency := time.Since(start)
		statusCode := c.Writer.Status()
		clientIP := c.ClientIP()
		method := c.Request.Method

		// Prepare log fields for structured logging.
		logFields := []zap.Field{
			zap.String("method", method),
			zap.String("path", path),
			zap.Int("status_code", statusCode),
			zap.Duration("latency", latency),
			zap.String("client_ip", clientIP),
		}

		// Add query parameters if they exist.
		if query != "" {
			logFields = append(logFields, zap.String("query", query))
		}

		// Add errors if any occurred and were added to Gin's context.
		if len(c.Errors) > 0 {
			// c.Errors.String() concatenates all error messages.
			// For more structured error logging, you might iterate c.Errors if they are of a specific type.
			logFields = append(logFields, zap.String("gin_errors", c.Errors.String()))
		}

		// Log with different levels based on status code.
		if statusCode >= http.StatusInternalServerError { // 500 and above
			logger.Error("Incoming Request", logFields...)
		} else if statusCode >= http.StatusBadRequest { // 400 to 499
			logger.Warn("Incoming Request", logFields...)
		} else { // 1xx, 2xx, 3xx
			logger.Info("Incoming Request", logFields...)
		}
	}
}
