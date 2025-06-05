package models

// CreateVaultRequest represents the request body for creating a new vault.
type CreateVaultRequest struct {
	Name        string   `json:"name" binding:"required"`
	Description string   `json:"description,omitempty"`
	Tags        []string `json:"tags,omitempty"`
}

// UpdateVaultRequest represents the request body for updating an existing vault.
// Pointers are used to distinguish between empty values (e.g. clear description) and fields not provided for update.
type UpdateVaultRequest struct {
	Name        *string  `json:"name,omitempty"`
	Description *string  `json:"description,omitempty"` // Use pointer to allow clearing the description
	Tags        *[]string `json:"tags,omitempty"`       // Use pointer to allow replacing tags, even with an empty list
}

// ShareVaultRequest represents the request body for sharing a vault.
type ShareVaultRequest struct {
	UserIDs         []string `json:"userIds" binding:"required"`         // User IDs to share with
	PermissionLevel string   `json:"permissionLevel" binding:"required"` // e.g., "read", "write"
}

// CreateSecretRequest represents the request body for creating a new secret.
type CreateSecretRequest struct {
	Name  string `json:"name" binding:"required"`
	Type  string `json:"type" binding:"required"`  // e.g., "secret", "certificate", "key_value"
	Value string `json:"value" binding:"required"` // Plain text value
	// ExpiresAt *time.Time `json:"expiresAt,omitempty"` // To be handled by models.Secret.ExpiresAt directly if needed
}

// UpdateSecretRequest represents the request body for updating an existing secret.
// Pointers are used to distinguish between empty values and fields not provided.
type UpdateSecretRequest struct {
	Name  *string `json:"name,omitempty"`
	Type  *string `json:"type,omitempty"`
	Value *string `json:"value,omitempty"` // Plain text value, if provided to update
	// ExpiresAt **time.Time `json:"expiresAt,omitempty"` // Complex case for optional time update;
	// For now, assume models.Secret.ExpiresAt is updated if Value is part of request, or handle separately.
	// A simpler approach for ExpiresAt might be a separate method or by requiring it if it's to be changed.
}
