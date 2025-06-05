package middleware

import (
	"net/http"
	"runtime/debug" // For logging stack trace

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// RecoveryMiddleware returns a gin.HandlerFunc (middleware)
// that recovers from any panics within a handler, logs the panic with a stack trace,
// and returns a generic 500 Internal Server Error response to the client.
// This prevents the server from crashing due to unhandled panics.
func RecoveryMiddleware(logger *zap.Logger) gin.HandlerFunc {
	if logger == nil {
		// Fallback or panic if logger is critical.
		panic("RecoveryMiddleware requires a non-nil zap.Logger instance")
	}
	return func(c *gin.Context) {
		defer func() {
			if err := recover(); err != nil {
				// Log panic error with stack trace.
				// zap.Stack("stacktrace") captures the stack trace of the current goroutine.
				// For panics, debug.Stack() provides the stack trace of the panicking goroutine.
				logger.Error("Panic recovered",
					zap.Any("error", err), // Log the panic error itself
					zap.String("stacktrace", string(debug.Stack())), // Log the stack trace
					zap.String("path", c.Request.URL.Path), // Include request path for context
					zap.String("method", c.Request.Method),
				)

				// If the response hasn't been written yet, send a generic 500 error.
				// This check prevents "multiple response.WriteHeader calls" errors.
				if !c.Writer.Written() {
					// Using gin.H for a simple JSON response.
					// Could use a predefined ErrorResponse struct if available and desired.
					c.JSON(http.StatusInternalServerError, gin.H{
						"error":   "Internal Server Error",
						"message": "The server encountered an unexpected condition which prevented it from fulfilling the request.",
					})
				}

				// Abort the request to prevent any further processing by other handlers
				// after a panic has occurred and been handled.
				c.Abort()
			}
		}()

		// Call the next handler in the chain.
		// If a panic occurs in a downstream handler, the defer function above will catch it.
		c.Next()
	}
}
