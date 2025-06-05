package models

// CreateVaultRequest represents the request body for creating a new vault.
type CreateVaultRequest struct {
	Name        string   `json:"name" binding:"required"`
	Description string   `json:"description,omitempty"` // Optional
	Tags        []string `json:"tags,omitempty"`        // Optional
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
	UserIDs         []string `json:"userIds" binding:"required,gt=0,dive,required"` // UserIDs array must be present, have at least 1 item, and each item (UID) must be present (not empty string)
	PermissionLevel string   `json:"permissionLevel" binding:"required,oneof=read write"` // Must be 'read' or 'write'
}

// CreateSecretRequest represents the request body for creating a new secret.
type CreateSecretRequest struct {
	Name  string `json:"name" binding:"required"`
	Type  string `json:"type" binding:"required"`  // e.g., "secret", "certificate", "key_value"
	Value string `json:"value" binding:"required"` // Plain text value; content validation (e.g. length) could be added with "validate"
	// ExpiresAt *time.Time `json:"expiresAt,omitempty"` // Optional, handled by models.Secret field directly
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
