package middleware

import (
	"log"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"vaultify-backend-go/internal/config" // To get CLIENT_URL for AllowOrigins
)

// CORSMiddleware configures Cross-Origin Resource Sharing (CORS) for the application.
// It allows requests from the CLIENT_URL specified in the application configuration
// and defines common HTTP methods and headers.
func CORSMiddleware(appConfig *config.Config) gin.HandlerFunc {
	if appConfig == nil || appConfig.ClientURL == "" {
		// This is a critical configuration error. The application might not work as expected
		// with clients if CORS is not properly configured.
		log.Fatal("CRITICAL_ERROR: appConfig or appConfig.ClientURL is not configured for CORSMiddleware. CORS will likely fail or be too permissive.")
		// Fallback to a very restrictive or default permissive CORS policy, or panic.
		// For safety, panic is better than a misconfigured permissive policy.
		panic("ClientURL for CORS is not configured")
	}

	return cors.New(cors.Config{
		// AllowOrigins specifies a list of origins that are allowed to make cross-origin requests.
		// Using appConfig.ClientURL makes this configurable.
		// For multiple origins, appConfig.ClientURL could be a comma-separated list parsed here,
		// or AllowOriginFunc could be used for more dynamic logic.
		AllowOrigins: []string{appConfig.ClientURL},

		// AllowMethods specifies which methods are allowed when accessing the resource.
		AllowMethods: []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},

		// AllowHeaders specifies which headers are allowed in the actual request.
		// "Authorization" is crucial for token-based auth.
		AllowHeaders: []string{"Origin", "Content-Type", "Accept", "Authorization", "X-Requested-With"},

		// ExposeHeaders indicates which headers are safe to expose to the API of a CORS API specification.
		ExposeHeaders: []string{"Content-Length"},

		// AllowCredentials indicates whether the request can include user credentials like cookies, HTTP authentication or client side SSL certificates.
		AllowCredentials: true,

		// MaxAge indicates how long the results of a preflight request can be cached.
		MaxAge: 12 * time.Hour, // Sensible default, adjust as needed.

		// AllowWildcard allows wildcard matching for origins. Be cautious with this.
		// AllowWildcard: true, // Generally not recommended for specific client URLs.

		// AllowBrowserExtensions allows browser extensions to make cross-origin requests.
		// AllowBrowserExtensions: true,

		// AllowWebSockets allows WebSocket requests.
		// AllowWebSockets: true,

		// AllowFiles allows file:// origins.
		// AllowFiles: true,
	})
}
