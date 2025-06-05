package api

import (
	"net/http"
	"log" // For logging unexpected issues

	"github.com/gin-gonic/gin"
	"vaultify-backend-go/internal/core"
	"vaultify-backend-go/internal/models" // Required for models.User response type
)

// AuthHandler handles authentication related API endpoints.
type AuthHandler struct {
	userService core.UserService
}

// NewAuthHandler creates a new AuthHandler.
func NewAuthHandler(us core.UserService) *AuthHandler {
	return &AuthHandler{userService: us}
}

// InitializeUserProfile handles the POST /api/v1/users/initialize endpoint.
// This endpoint is intended to be called by a client after a Firebase authentication event (login/signup)
// to ensure that a corresponding user profile exists in the application's database.
// It relies on an authentication middleware to validate the Firebase ID token and extract user information
// into the Gin context.
func (h *AuthHandler) InitializeUserProfile(c *gin.Context) {
	// --- Retrieve user information from Gin context (populated by auth middleware) ---
	// Expected context keys:
	// - "userID": string, the Firebase UID of the authenticated user.
	// - "userEmail": string, the email of the authenticated user.
	// - "userDisplayName": string, the display name of the authenticated user.
	// - "userPhotoURL": string, the photo URL of the authenticated user.

	rawUserID, exists := c.Get("userID")
	if !exists {
		log.Println("InitializeUserProfile Error: userID not found in context. Auth middleware might not have run or failed.")
		c.JSON(http.StatusUnauthorized, ErrorResponse{Error: "Authentication error: User ID not found in context"})
		return
	}
	firebaseUserID, ok := rawUserID.(string)
	if !ok || firebaseUserID == "" {
		log.Printf("InitializeUserProfile Error: userID in context is not a valid string or is empty. Value: %v", rawUserID)
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "Invalid user ID format in context"})
		return
	}

	rawUserEmail, exists := c.Get("userEmail")
	if !exists { // Email might be optional in some Firebase configs, but likely required for our app
		log.Printf("InitializeUserProfile Warning: userEmail not found in context for userID: %s.", firebaseUserID)
		// Decide if email is strictly required. For GetOrCreate, it typically is.
		// c.JSON(http.StatusBadRequest, ErrorResponse{Error: "User email not found in context"})
		// return
	}
	email, _ := rawUserEmail.(string) // Allow empty if not strictly required or handle error

	// DisplayName and PhotoURL can be optional
	rawDisplayName, _ := c.Get("userDisplayName")
	displayName, _ := rawDisplayName.(string)

	rawPhotoURL, _ := c.Get("userPhotoURL")
	photoURL, _ := rawPhotoURL.(string)

	// --- Call the UserService to get or create the user profile ---
	// The UserService's GetOrCreate method encapsulates the logic of checking existence and creating if new.
	user, created, err := h.userService.GetOrCreate(c.Request.Context(), firebaseUserID, email, displayName, photoURL)
	if err != nil {
		// TODO: Implement more sophisticated error mapping from service errors to HTTP status codes.
		// For example, a database error should be 500, a validation error (if any) 400.
		log.Printf("InitializeUserProfile Error: userService.GetOrCreate failed for userID %s: %v", firebaseUserID, err)
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "Failed to initialize user profile", Details: err.Error()})
		return
	}

	// --- Respond based on whether the user was newly created or already existed ---
	if created {
		log.Printf("User profile created for userID: %s", firebaseUserID)
		c.JSON(http.StatusCreated, user) // Return the full user model (as defined by models.User)
	} else {
		log.Printf("User profile already existed for userID: %s", firebaseUserID)
		c.JSON(http.StatusOK, user) // Return the full user model
	}
}
