package api

import (
	"errors"
	"net/http"
	"strings" // For parsing query params like tags
	"log"     // For logging errors or unexpected issues

	"github.com/gin-gonic/gin"
	"vaultify-backend-go/internal/core"
	"vaultify-backend-go/internal/models"
	// "github.com/go-playground/validator/v10" // For future request validation
)

// VaultHandler handles API endpoints related to vaults.
type VaultHandler struct {
	vaultService core.VaultService
}

// NewVaultHandler creates a new VaultHandler.
func NewVaultHandler(vs core.VaultService) *VaultHandler {
	return &VaultHandler{vaultService: vs}
}

// mapVaultErrorToStatus maps errors from core.VaultService to HTTP status codes and ErrorResponse.
func mapVaultErrorToStatus(c *gin.Context, err error) {
	var statusCode int
	var errResponse ErrorResponse

	switch {
	case errors.Is(err, core.ErrVaultNotFound):
		statusCode = http.StatusNotFound
		errResponse = ErrorResponse{Error: core.ErrVaultNotFound.Error()}
	case errors.Is(err, core.ErrForbiddenAccess):
		statusCode = http.StatusForbidden
		errResponse = ErrorResponse{Error: core.ErrForbiddenAccess.Error()}
	case errors.Is(err, core.ErrVaultLimitReached):
		// Using 402 Payment Required, though 403 Forbidden or 400 Bad Request might also be applicable
		// depending on how "limits" are presented to the user (e.g., as a hard paywall or a usage quota).
		statusCode = http.StatusPaymentRequired
		errResponse = ErrorResponse{Error: core.ErrVaultLimitReached.Error()}
	case errors.Is(err, core.ErrUserNotFound): // If vault service uses/propagates this
		statusCode = http.StatusNotFound
		errResponse = ErrorResponse{Error: "User not found", Details: err.Error()}
	case errors.Is(err, core.ErrCannotShareWithSelf):
		statusCode = http.StatusBadRequest
		errResponse = ErrorResponse{Error: core.ErrCannotShareWithSelf.Error()}
	case errors.Is(err, core.ErrUserAlreadyHasAccess): // Assuming this error exists in core
		statusCode = http.StatusConflict // 409 Conflict is often suitable for "already exists" type errors
		errResponse = ErrorResponse{Error: core.ErrUserAlreadyHasAccess.Error()}
	case errors.Is(err, core.ErrShareTargetUserNotFound): // Assuming this error exists in core
		statusCode = http.StatusBadRequest // Or StatusNotFound if specifically for the target user
		errResponse = ErrorResponse{Error: core.ErrShareTargetUserNotFound.Error()}
	case errors.Is(err, core.ErrInvalidPermissionLevel):
		statusCode = http.StatusBadRequest
		errResponse = ErrorResponse{Error: core.ErrInvalidPermissionLevel.Error()}
	// Add other specific errors from VaultService here
	// Example:
	// case errors.Is(err, core.ErrSpecificVaultActionFailed):
	//    statusCode = http.StatusBadRequest
	//    errResponse = ErrorResponse{Error: "Specific action failed", Details: err.Error()}
	default:
		// Check for validation errors if using a library like go-playground/validator
		// var validationErrors validator.ValidationErrors
		// if errors.As(err, &validationErrors) {
		//     statusCode = http.StatusBadRequest
		//     errResponse = ErrorResponse{Error: "Validation failed", Details: validationErrors.Error()}
		// } else {
		log.Printf("Internal Server Error: %v", err) // Log the actual error for server-side review
		statusCode = http.StatusInternalServerError
		errResponse = ErrorResponse{Error: "An unexpected internal server error occurred."}
		// }
	}
	c.JSON(statusCode, errResponse)
}

// CreateVault handles POST /vaults
func (h *VaultHandler) CreateVault(c *gin.Context) {
	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, ErrorResponse{Error: "User ID not found in context"})
		return
	}

	var req models.CreateVaultRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "Invalid request payload", Details: err.Error()})
		return
	}

	// TODO: Add request payload validation (e.g., Name is required, tags format if any)
	// Example: validate := validator.New()
	// if err := validate.Struct(&req); err != nil {
	//     mapVaultErrorToStatus(c, err) // Assuming mapVaultErrorToStatus handles validation errors
	//     return
	// }


	createdVault, err := h.vaultService.CreateVault(c.Request.Context(), userID.(string), req)
	if err != nil {
		mapVaultErrorToStatus(c, err)
		return
	}
	c.JSON(http.StatusCreated, createdVault)
}

// GetVault handles GET /vaults/:vaultId
func (h *VaultHandler) GetVault(c *gin.Context) {
	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, ErrorResponse{Error: "User ID not found in context"})
		return
	}
	vaultID := c.Param("vaultId")
	if vaultID == "" {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "Vault ID is required"})
		return
	}

	vault, err := h.vaultService.GetVaultByID(c.Request.Context(), userID.(string), vaultID)
	if err != nil {
		mapVaultErrorToStatus(c, err)
		return
	}
	c.JSON(http.StatusOK, vault)
}

// ListVaults handles GET /vaults
func (h *VaultHandler) ListVaults(c *gin.Context) {
	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, ErrorResponse{Error: "User ID not found in context"})
		return
	}

	// TODO: Implement robust pagination and filtering
	// For now, passing empty map for paginationParams
	paginationParams := make(map[string]string)
	if c.Query("limit") != "" {
		paginationParams["limit"] = c.Query("limit")
	}
	if c.Query("startAfter") != "" {
		paginationParams["startAfter"] = c.Query("startAfter")
	}
	// tagsParam := c.Query("tags") // Example: ?tags=work,important
	// if tagsParam != "" {
	//     paginationParams["tags"] = tagsParam // Service needs to handle splitting and querying tags
	// }


	vaults, err := h.vaultService.ListVaults(c.Request.Context(), userID.(string), paginationParams)
	if err != nil {
		mapVaultErrorToStatus(c, err)
		return
	}
	c.JSON(http.StatusOK, vaults)
}

// UpdateVault handles PUT /vaults/:vaultId
func (h *VaultHandler) UpdateVault(c *gin.Context) {
	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, ErrorResponse{Error: "User ID not found in context"})
		return
	}
	vaultID := c.Param("vaultId")
	if vaultID == "" {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "Vault ID is required"})
		return
	}

	var req models.UpdateVaultRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "Invalid request payload", Details: err.Error()})
		return
	}

	// TODO: Add request payload validation if needed (e.g. if certain fields become non-optional)

	updatedVault, err := h.vaultService.UpdateVault(c.Request.Context(), userID.(string), vaultID, req)
	if err != nil {
		mapVaultErrorToStatus(c, err)
		return
	}
	c.JSON(http.StatusOK, updatedVault)
}

// DeleteVault handles DELETE /vaults/:vaultId
func (h *VaultHandler) DeleteVault(c *gin.Context) {
	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, ErrorResponse{Error: "User ID not found in context"})
		return
	}
	vaultID := c.Param("vaultId")
	if vaultID == "" {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "Vault ID is required"})
		return
	}

	err := h.vaultService.DeleteVault(c.Request.Context(), userID.(string), vaultID)
	if err != nil {
		mapVaultErrorToStatus(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

// ShareVault handles POST /vaults/:vaultId/share
func (h *VaultHandler) ShareVault(c *gin.Context) {
	ownerID, exists := c.Get("userID") // The current user is the owner performing the share action
	if !exists {
		c.JSON(http.StatusUnauthorized, ErrorResponse{Error: "User ID (owner) not found in context"})
		return
	}
	vaultID := c.Param("vaultId")
	if vaultID == "" {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "Vault ID is required"})
		return
	}

	var req models.ShareVaultRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "Invalid request payload", Details: err.Error()})
		return
	}

	// TODO: Validate request payload: UserIDs not empty, PermissionLevel is valid ("read", "write")
	if len(req.UserIDs) == 0 {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "UserIDs cannot be empty for sharing"})
		return
	}
	if req.PermissionLevel != "read" && req.PermissionLevel != "write" {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "Invalid permission level. Must be 'read' or 'write'."})
		return
	}


	err := h.vaultService.ShareVault(c.Request.Context(), ownerID.(string), vaultID, req)
	if err != nil {
		mapVaultErrorToStatus(c, err)
		return
	}
	// Consider returning the updated vault or just a success message
	c.JSON(http.StatusOK, SuccessResponse{Message: "Vault shared successfully"})
}

// UpdateShareRequest defines the expected request body for updating share permissions.
type UpdateShareRequest struct {
	PermissionLevel string `json:"permissionLevel" binding:"required"`
}

// UpdateShare handles PUT /vaults/:vaultId/share/:targetUserId
func (h *VaultHandler) UpdateShare(c *gin.Context) {
	ownerID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, ErrorResponse{Error: "User ID (owner) not found in context"})
		return
	}
	vaultID := c.Param("vaultId")
	targetUserID := c.Param("targetUserId")
	if vaultID == "" || targetUserID == "" {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "Vault ID and Target User ID are required"})
		return
	}

	var req UpdateShareRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "Invalid request payload", Details: err.Error()})
		return
	}

	if req.PermissionLevel != "read" && req.PermissionLevel != "write" {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "Invalid permission level. Must be 'read' or 'write'."})
		return
	}


	err := h.vaultService.UpdateSharePermissions(c.Request.Context(), ownerID.(string), vaultID, targetUserID, req.PermissionLevel)
	if err != nil {
		mapVaultErrorToStatus(c, err)
		return
	}
	c.JSON(http.StatusOK, SuccessResponse{Message: "Vault share permissions updated successfully"})
}

// RemoveShare handles DELETE /vaults/:vaultId/share/:targetUserId
func (h *VaultHandler) RemoveShare(c *gin.Context) {
	ownerID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, ErrorResponse{Error: "User ID (owner) not found in context"})
		return
	}
	vaultID := c.Param("vaultId")
	targetUserID := c.Param("targetUserId")
	if vaultID == "" || targetUserID == "" {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "Vault ID and Target User ID are required"})
		return
	}

	err := h.vaultService.RemoveShare(c.Request.Context(), ownerID.(string), vaultID, targetUserID)
	if err != nil {
		mapVaultErrorToStatus(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}
