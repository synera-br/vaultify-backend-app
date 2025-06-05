package middleware

import (
	"context"
	"net/http"
	"strings"
	"log" // For logging critical errors or unexpected behavior

	"firebase.google.com/go/v4/auth"
	"github.com/gin-gonic/gin"
	// To avoid potential import cycles with internal/api, ErrorResponse is defined locally.
	// If ErrorResponse were in a common package like internal/common/types, that could be imported.
)

// ErrorResponse is a local definition for sending standardized error messages.
// It mirrors the one in internal/api/dto_models.go to avoid import cycles.
type ErrorResponse struct {
	Error   string `json:"error"`
	Details string `json:"details,omitempty"`
}

// AuthMiddleware provides Gin middleware for Firebase token authentication.
type AuthMiddleware struct {
	firebaseAuthClient *auth.Client
}

// NewAuthMiddleware creates a new AuthMiddleware instance.
// It panics if the firebaseAuthClient is nil, as this is a critical setup dependency.
func NewAuthMiddleware(fbAuthClient *auth.Client) *AuthMiddleware {
	if fbAuthClient == nil {
		// Using panic here because a nil auth client is a programmer error during setup
		// and the application cannot function correctly without it for authenticated routes.
		log.Fatal("CRITICAL_ERROR: Firebase Auth client is not initialized for AuthMiddleware. Ensure db.InitFirestore() and db.GetFirebaseAuthClient() are called and succeed before initializing middleware.")
		panic("Firebase Auth client is not initialized for AuthMiddleware")
	}
	return &AuthMiddleware{firebaseAuthClient: fbAuthClient}
}

// VerifyToken is a Gin middleware handler function that verifies a Firebase ID token
// from the Authorization header. If valid, it sets user information in the Gin context.
func (m *AuthMiddleware) VerifyToken() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, ErrorResponse{Error: "Authorization header is required"})
			return
		}

		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") { // Use EqualFold for case-insensitive "bearer"
			c.AbortWithStatusJSON(http.StatusUnauthorized, ErrorResponse{Error: "Authorization header format must be 'Bearer {token}'"})
			return
		}
		idToken := parts[1]

		// Use c.Request.Context() for the VerifyIDToken call as it's request-scoped.
		// context.Background() is generally for utility or server-lifetime contexts.
		token, err := m.firebaseAuthClient.VerifyIDToken(c.Request.Context(), idToken)
		if err != nil {
			log.Printf("AuthMiddleware: Error verifying Firebase ID token: %v", err)
			// Provide a generic error message to the client for security.
			// Specific error details are logged server-side.
			c.AbortWithStatusJSON(http.StatusUnauthorized, ErrorResponse{Error: "Invalid or expired authentication token"})
			return
		}

		// Token is valid. Set user information in context for downstream handlers.
		c.Set("userID", token.UID)

		// Extract standard claims that might be useful for user profile initialization or other purposes.
		// Firebase populates these from the token if they exist.
		if email, ok := token.Claims["email"].(string); ok {
			c.Set("userEmail", email)
		}
		// 'name' is a common claim for display name in Firebase ID tokens.
		if name, ok := token.Claims["name"].(string); ok {
			c.Set("userDisplayName", name)
		}
		// 'picture' is a common claim for photo URL.
		if picture, ok := token.Claims["picture"].(string); ok {
			c.Set("userPhotoURL", picture)
		}

		// For a cleaner way to pass multiple claims, consider a struct:
		/*
		   type AuthenticatedUserClaims struct {
		       UID         string
		       Email       string
		       DisplayName string
		       PhotoURL    string
		       // Add other claims as needed
		   }
		   claims := AuthenticatedUserClaims{
		       UID: token.UID,
		       Email: c.GetString("userEmail"), // Or get directly from token.Claims
		       DisplayName: c.GetString("userDisplayName"),
		       PhotoURL: c.GetString("userPhotoURL"),
		   }
		   c.Set("authenticatedUserClaims", claims)
		*/

		c.Next() // Proceed to the next handler in the chain.
	}
}
