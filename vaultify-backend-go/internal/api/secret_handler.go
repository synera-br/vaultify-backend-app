package api

import (
	"errors"
	"net/http"
	"log"     // For logging errors or unexpected issues
	"time"    // For SecretDetailResponse

	"github.com/gin-gonic/gin"
	"vaultify-backend-go/internal/core"
	"vaultify-backend-go/internal/models"
	// "github.com/go-playground/validator/v10" // For future request validation
)

// SecretHandler handles API endpoints related to secrets within a vault.
type SecretHandler struct {
	secretService core.SecretService
}

// NewSecretHandler creates a new SecretHandler.
func NewSecretHandler(ss core.SecretService) *SecretHandler {
	return &SecretHandler{secretService: ss}
}

// mapSecretErrorToStatus maps errors from core.SecretService to HTTP status codes and ErrorResponse.
func mapSecretErrorToStatus(c *gin.Context, err error) {
	var statusCode int
	var errResponse ErrorResponse

	switch {
	case errors.Is(err, core.ErrSecretNotFound):
		statusCode = http.StatusNotFound
		errResponse = ErrorResponse{Error: core.ErrSecretNotFound.Error()}
	case errors.Is(err, core.ErrVaultNotFound): // Propagated from vault access check
		statusCode = http.StatusNotFound
		errResponse = ErrorResponse{Error: "Parent vault not found", Details: err.Error()}
	case errors.Is(err, core.ErrForbiddenAccess): // Propagated from vault access check
		statusCode = http.StatusForbidden
		errResponse = ErrorResponse{Error: core.ErrForbiddenAccess.Error()}
	case errors.Is(err, core.ErrEncryptionFailed):
		statusCode = http.StatusInternalServerError
		errResponse = ErrorResponse{Error: "Failed to encrypt secret data."} // Avoid exposing too many details
	case errors.Is(err, core.ErrDecryptionFailed):
		statusCode = http.StatusInternalServerError
		errResponse = ErrorResponse{Error: "Failed to decrypt secret data. Data may be corrupted or key is incorrect."}
	case errors.Is(err, core.ErrSecretLimitReached):
		statusCode = http.StatusPaymentRequired // Or 400/403 depending on policy
		errResponse = ErrorResponse{Error: core.ErrSecretLimitReached.Error()}
	case errors.Is(err, core.ErrInvalidEncryptionKey): // This is a server configuration issue
		log.Printf("Critical: Invalid Encryption Key configured: %v", err)
		statusCode = http.StatusInternalServerError
		errResponse = ErrorResponse{Error: "Server encryption configuration error. Please contact support."}
	// Add other specific errors from SecretService if any
	default:
		// var validationErrors validator.ValidationErrors
		// if errors.As(err, &validationErrors) {
		//     statusCode = http.StatusBadRequest
		//     errResponse = ErrorResponse{Error: "Validation failed", Details: validationErrors.Error()}
		// } else {
		log.Printf("Internal Server Error in SecretHandler: %v", err)
		statusCode = http.StatusInternalServerError
		errResponse = ErrorResponse{Error: "An unexpected internal server error occurred while processing your secret request."}
		// }
	}
	c.JSON(statusCode, errResponse)
}


// CreateSecret handles POST /vaults/:vaultId/secrets
func (h *SecretHandler) CreateSecret(c *gin.Context) {
	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, ErrorResponse{Error: "User ID not found in context"})
		return
	}
	vaultID := c.Param("vaultId")
	if vaultID == "" {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "Vault ID is required in path"})
		return
	}

	var req models.CreateSecretRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "Invalid request payload", Details: err.Error()})
		return
	}

	// TODO: Add comprehensive request payload validation
	if req.Name == "" || req.Value == "" || req.Type == "" {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "Validation failed: name, value, and type are required for secret"})
		return
	}

	createdSecret, err := h.secretService.CreateSecret(c.Request.Context(), userID.(string), vaultID, req)
	if err != nil {
		mapSecretErrorToStatus(c, err)
		return
	}
	// The service returns models.Secret which does not include the decrypted value.
	// It includes EncryptedValue, which we typically don't want to send back on create.
	// Let's create a response that mirrors the structure but omits EncryptedValue.
	response := models.Secret{ // Re-using models.Secret but be mindful of EncryptedValue
		ID:        createdSecret.ID,
		VaultID:   vaultID, // Add vaultID for context in response
		Name:      createdSecret.Name,
		Type:      createdSecret.Type,
		ExpiresAt: createdSecret.ExpiresAt,
		CreatedAt: createdSecret.CreatedAt,
		UpdatedAt: createdSecret.UpdatedAt,
		// EncryptedValue: "" // Explicitly omit or ensure JSON omitempty works if it were present
	}
	c.JSON(http.StatusCreated, response)
}

// GetSecret handles GET /vaults/:vaultId/secrets/:secretId
func (h *SecretHandler) GetSecret(c *gin.Context) {
	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, ErrorResponse{Error: "User ID not found in context"})
		return
	}
	vaultID := c.Param("vaultId")
	secretID := c.Param("secretId")
	if vaultID == "" || secretID == "" {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "Vault ID and Secret ID are required in path"})
		return
	}

	secretModel, decryptedValue, err := h.secretService.GetSecretByID(c.Request.Context(), userID.(string), vaultID, secretID)
	if err != nil {
		mapSecretErrorToStatus(c, err)
		return
	}

	response := SecretDetailResponse{ // Using the specific DTO for this endpoint
		ID:        secretModel.ID,
		VaultID:   vaultID, // from path param, or secretModel.VaultID if service populates it
		Name:      secretModel.Name,
		Type:      secretModel.Type,
		Value:     decryptedValue,
		ExpiresAt: secretModel.ExpiresAt,
		CreatedAt: secretModel.CreatedAt,
		UpdatedAt: secretModel.UpdatedAt,
	}
	c.JSON(http.StatusOK, response)
}

// ListSecrets handles GET /vaults/:vaultId/secrets
func (h *SecretHandler) ListSecrets(c *gin.Context) {
	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, ErrorResponse{Error: "User ID not found in context"})
		return
	}
	vaultID := c.Param("vaultId")
	if vaultID == "" {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "Vault ID is required in path"})
		return
	}

	// TODO: Implement robust pagination and filtering
	paginationParams := make(map[string]string)
	if c.Query("limit") != "" {
		paginationParams["limit"] = c.Query("limit")
	}
	if c.Query("startAfter") != "" {
		paginationParams["startAfter"] = c.Query("startAfter")
	}

	secrets, err := h.secretService.ListSecrets(c.Request.Context(), userID.(string), vaultID, paginationParams)
	if err != nil {
		mapSecretErrorToStatus(c, err)
		return
	}
	// The service returns []*models.Secret which should not have decrypted values.
	// They will have EncryptedValue. We should transform this to a list of responses
	// that explicitly omit EncryptedValue for security/hygiene.
	var responseSecrets []models.Secret // Re-using models.Secret, ensuring EncryptedValue is not exposed
	for _, s := range secrets {
		responseSecrets = append(responseSecrets, models.Secret{
			ID:        s.ID,
			VaultID:   vaultID, // Add vaultID for context
			Name:      s.Name,
			Type:      s.Type,
			ExpiresAt: s.ExpiresAt,
			CreatedAt: s.CreatedAt,
			UpdatedAt: s.UpdatedAt,
		})
	}
	c.JSON(http.StatusOK, responseSecrets)
}

// UpdateSecret handles PUT /vaults/:vaultId/secrets/:secretId
func (h *SecretHandler) UpdateSecret(c *gin.Context) {
	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, ErrorResponse{Error: "User ID not found in context"})
		return
	}
	vaultID := c.Param("vaultId")
	secretID := c.Param("secretId")
	if vaultID == "" || secretID == "" {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "Vault ID and Secret ID are required in path"})
		return
	}

	var req models.UpdateSecretRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "Invalid request payload", Details: err.Error()})
		return
	}

	// TODO: Add comprehensive request payload validation

	updatedSecret, err := h.secretService.UpdateSecret(c.Request.Context(), userID.(string), vaultID, secretID, req)
	if err != nil {
		mapSecretErrorToStatus(c, err)
		return
	}
	// Similar to CreateSecret, ensure response omits EncryptedValue
	response := models.Secret{
		ID:        updatedSecret.ID,
		VaultID:   vaultID,
		Name:      updatedSecret.Name,
		Type:      updatedSecret.Type,
		ExpiresAt: updatedSecret.ExpiresAt,
		CreatedAt: updatedSecret.CreatedAt,
		UpdatedAt: updatedSecret.UpdatedAt,
	}
	c.JSON(http.StatusOK, response)
}

// DeleteSecret handles DELETE /vaults/:vaultId/secrets/:secretId
func (h *SecretHandler) DeleteSecret(c *gin.Context) {
	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, ErrorResponse{Error: "User ID not found in context"})
		return
	}
	vaultID := c.Param("vaultId")
	secretID := c.Param("secretId")
	if vaultID == "" || secretID == "" {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "Vault ID and Secret ID are required in path"})
		return
	}

	err := h.secretService.DeleteSecret(c.Request.Context(), userID.(string), vaultID, secretID)
	if err != nil {
		mapSecretErrorToStatus(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}
