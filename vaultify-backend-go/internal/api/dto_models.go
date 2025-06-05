package api

// ErrorResponse is a generic structure for returning errors via API.
type ErrorResponse struct {
	Error   string `json:"error"`             // A high-level error message or code
	Details string `json:"details,omitempty"` // More specific details about the error, if available
}

// SuccessResponse is a generic structure for simple success messages.
type SuccessResponse struct {
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

// SecretDetailResponse is used for GET /vaults/:vaultId/secrets/:secretId to include the decrypted value.
type SecretDetailResponse struct {
	ID        string     `json:"id"`
	VaultID   string     `json:"vaultId"`
	Name      string     `json:"name"`
	Type      string     `json:"type"`
	Value     string     `json:"value"` // Decrypted value
	ExpiresAt *time.Time `json:"expiresAt,omitempty"`
	CreatedAt time.Time  `json:"createdAt"`
	UpdatedAt time.Time  `json:"updatedAt"`
}

// Consider adding other common Data Transfer Objects (DTOs) here if they are
// primarily used for API request/response shaping and don't fit directly
// into the `internal/models` domain or request models.
// For example, specific response structures that aggregate data from multiple models.
// For now, ErrorResponse and SuccessResponse are common starting points.
