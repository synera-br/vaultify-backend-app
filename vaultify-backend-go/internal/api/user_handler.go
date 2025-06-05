package api

import (
	"net/http"
	"log" // For logging
	"errors" // For errors.Is

	"github.com/gin-gonic/gin"
	"vaultify-backend-go/internal/core" // For core services and errors like core.ErrUserNotFound
	// "vaultify-backend-go/internal/db" // For db.ErrNotFound if service passes it through directly
)

// UserHandler handles user-profile related API endpoints.
type UserHandler struct {
	userService core.UserService
}

// NewUserHandler creates a new UserHandler.
func NewUserHandler(us core.UserService) *UserHandler {
	return &UserHandler{userService: us}
}

// GetCurrentUserProfile handles the GET /api/v1/users/me endpoint.
// It retrieves the profile of the currently authenticated user.
func (h *UserHandler) GetCurrentUserProfile(c *gin.Context) {
	// --- Retrieve userID from Gin context (populated by auth middleware) ---
	rawUserID, exists := c.Get("userID")
	if !exists {
		log.Println("GetCurrentUserProfile Error: userID not found in context. Auth middleware might not have run or failed.")
		c.JSON(http.StatusUnauthorized, ErrorResponse{Error: "Authentication error: User ID not found in context"})
		return
	}
	firebaseUserID, ok := rawUserID.(string)
	if !ok || firebaseUserID == "" {
		log.Printf("GetCurrentUserProfile Error: userID in context is not a valid string or is empty. Value: %v", rawUserID)
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "Invalid user ID format in context"})
		return
	}

	// --- Call the UserService to get the user profile ---
	user, err := h.userService.GetByID(c.Request.Context(), firebaseUserID)
	if err != nil {
		// TODO: Standardize error handling and mapping.
		// Check for specific errors from the service layer, e.g., ErrUserNotFound.
		// This requires core.ErrUserNotFound to be defined and returned by the service.
		// For now, using errors.Is with a placeholder core.ErrUserNotFound (if it were defined there).
		// If core.ErrUserNotFound is not yet defined, this check won't work as intended.
		// A temporary string check as a placeholder:
		if err.Error() == "user not found" || (errors.Is(err, core.ErrUserNotFound)) { // core.ErrUserNotFound needs to be defined
			log.Printf("GetCurrentUserProfile: User profile not found for userID %s.", firebaseUserID)
			c.JSON(http.StatusNotFound, ErrorResponse{Error: "User profile not found"})
			return
		}

		// For other errors
		log.Printf("GetCurrentUserProfile Error: userService.GetByID failed for userID %s: %v", firebaseUserID, err)
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "Failed to retrieve user profile", Details: err.Error()})
		return
	}

	// --- Respond with the user profile ---
	// The user model (models.User) is returned directly.
	// Ensure it's properly tagged for JSON and doesn't expose sensitive data not meant for the client.
	c.JSON(http.StatusOK, user)
}
