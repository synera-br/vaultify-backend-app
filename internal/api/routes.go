package api

import (
	// "firebase.google.com/go/v4/auth" // Not directly used in this simplified version, but might be needed for more complex scenarios
	"github.com/gin-gonic/gin"
)

// RegisterRoutes sets up the API routes for the v1 group.
// It requires the encryption key to be passed to handlers that need it.
func RegisterRoutes(router *gin.RouterGroup, encryptionKey []byte /*, fbAuth *auth.Client - pass if needed directly by handlers */) {
	// Example encrypted endpoint
	// POST /api/v1/process
	// Body: {"payload": "Base64EncodedEncryptedData"}
	router.POST("/process", HandleEncryptedData(encryptionKey))

	// Example simple authenticated GET endpoint
	// GET /api/v1/user/info
	router.GET("/user/info", GetUserDataHandler)

	// Add more routes here as needed
}
