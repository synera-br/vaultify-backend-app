package core

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"time"

	"vaultify-backend-go/internal/config" // To get ENCRYPTION_KEY
	"vaultify-backend-go/internal/db"
	"vaultify-backend-go/internal/models"
)

// Custom errors for the SecretService
var (
	ErrSecretNotFound         = errors.New("secret not found")
	ErrEncryptionFailed       = errors.New("failed to encrypt secret value")
	ErrDecryptionFailed       = errors.New("failed to decrypt secret value")
	ErrSecretLimitReached     = errors.New("secret limit reached for the current plan or vault")
	ErrInvalidEncryptionKey   = errors.New("invalid encryption key loaded")
	// ErrForbiddenAccess is already defined in vault_service.go, can be reused or defined commonly
	// ErrVaultNotFound is already defined in vault_service.go, can be reused or defined commonly
)

// secretService implements the SecretService interface.
type secretService struct {
	secretRepo        db.SecretRepository
	vaultRepo         db.VaultRepository // To check vault access permissions
	userRepo          db.UserRepository  // For plan limits, if applicable (not used in this snippet yet)
	encryptionService EncryptionService
	auditService      AuditService
	encryptionKey     []byte // Decoded encryption key from config
}

// NewSecretService creates a new SecretService instance.
// It also decodes the encryption key from the application configuration.
func NewSecretService(
	sr db.SecretRepository,
	vr db.VaultRepository,
	ur db.UserRepository, // Included for future plan checks on secrets
	es EncryptionService,
	as AuditService,
	appConfig *config.Config,
) (SecretService, error) {
	if appConfig == nil || appConfig.EncryptionKey == "" {
		return nil, fmt.Errorf("%w: encryption key is missing from application configuration", ErrInvalidEncryptionKey)
	}

	key, err := base64.StdEncoding.DecodeString(appConfig.EncryptionKey)
	if err != nil {
		return nil, fmt.Errorf("%w: failed to decode base64 encryption key: %v", ErrInvalidEncryptionKey, err)
	}
	if len(key) != 32 { // AES-256 requires a 32-byte key
		return nil, fmt.Errorf("%w: key length must be 32 bytes for AES-256, but got %d bytes", ErrInvalidEncryptionKey, len(key))
	}

	return &secretService{
		secretRepo:        sr,
		vaultRepo:         vr,
		userRepo:          ur,
		encryptionService: es,
		auditService:      as,
		encryptionKey:     key,
	}, nil
}

// checkVaultAccess is a helper function to verify if a user has the required permission level for a vault.
// It returns the vault model if access is granted, or an error otherwise.
func (s *secretService) checkVaultAccess(ctx context.Context, userID, vaultID string, requiredPermissionLevel string) (*models.Vault, error) {
	if s.vaultRepo == nil {
		return nil, errors.New("secretService: vaultRepo not initialized")
	}

	vault, err := s.vaultRepo.GetByID(ctx, vaultID)
	if err != nil {
		// Assuming GetByID returns a specific error for not found, e.g., db.ErrVaultNotFound
		// For now, wrap it.
		return nil, fmt.Errorf("failed to retrieve vault '%s': %w", vaultID, err)
	}
	if vault == nil {
		// This path should ideally be covered if GetByID returns a specific ErrNotFound
		return nil, ErrVaultNotFound // Using error from vault_service for consistency
	}

	if vault.OwnerID == userID {
		return vault, nil // Owner has all permissions
	}

	permission, ok := vault.SharedWith[userID]
	if !ok {
		return nil, fmt.Errorf("%w: user '%s' does not have access to vault '%s'", ErrForbiddenAccess, userID, vaultID)
	}

	// Check if the granted permission is sufficient for the required action
	// "write" permission implies "read" permission
	if requiredPermissionLevel == "read" && (permission == "read" || permission == "write") {
		return vault, nil
	}
	if requiredPermissionLevel == "write" && permission == "write" {
		return vault, nil
	}

	return nil, fmt.Errorf("%w: user '%s' has '%s' permission, but requires '%s' for vault '%s'", ErrForbiddenAccess, userID, permission, requiredPermissionLevel, vaultID)
}


// CreateSecret creates a new encrypted secret within a vault.
func (s *secretService) CreateSecret(ctx context.Context, userID, vaultID string, req models.CreateSecretRequest) (*models.Secret, error) {
	if s.secretRepo == nil || s.encryptionService == nil || s.auditService == nil {
		return nil, errors.New("secretService: component not initialized")
	}

	_, err := s.checkVaultAccess(ctx, userID, vaultID, "write")
	if err != nil {
		return nil, err // Handles vault not found or forbidden access
	}

	// TODO: Implement secret limit checks based on user's plan or vault's capacity.
	// Example:
	// user, err := s.userRepo.GetByID(ctx, userID) // If plan is user-based
	// if err != nil { return nil, fmt.Errorf("failed to get user for plan check: %w", err) }
	// currentSecretCount, err := s.secretRepo.CountByVaultID(ctx, vaultID) // Or CountByOwnerID if global limit
	// if err != nil { return nil, fmt.Errorf("failed to count secrets: %w", err) }
	// if err := s.checkSecretLimit(user.Plan, currentSecretCount); err != nil { // checkSecretLimit helper
	//     return nil, err
	// }

	encryptedValue, err := s.encryptionService.Encrypt(req.Value, s.encryptionKey)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrEncryptionFailed, err)
	}

	newSecret := &models.Secret{
		// VaultID is not stored in the Secret model itself in Firestore,
		// as it's part of the path (subcollection). But it's good for context here.
		// VaultID:      vaultID,
		Name:           req.Name,
		Type:           req.Type,
		EncryptedValue: encryptedValue,
		CreatedAt:      time.Now().UTC(),
		UpdatedAt:      time.Now().UTC(),
		// ExpiresAt: // Handle if CreateSecretRequest includes ExpiresAt
	}

	secretID, err := s.secretRepo.Create(ctx, vaultID, newSecret)
	if err != nil {
		return nil, fmt.Errorf("failed to create secret in repository: %w", err)
	}
	newSecret.ID = secretID // Set the ID returned by the repository
	newSecret.VaultID = vaultID // For returning context, not for storage in this specific field

	// Audit Log
	auditLogEntry := models.AuditLog{
		UserID:     userID,
		Action:     "SECRET_CREATE",
		TargetType: "SECRET",
		TargetID:   newSecret.ID,
		Details: map[string]interface{}{
			"vault_id":    vaultID,
			"secret_name": newSecret.Name,
			"secret_type": newSecret.Type,
		},
		Timestamp: time.Now().UTC(),
	}
	if auditErr := s.auditService.CreateAuditLog(ctx, auditLogEntry); auditErr != nil {
		fmt.Printf("Warning: failed to create audit log for SECRET_CREATE (secretID: %s): %v\n", newSecret.ID, auditErr)
	}

	// Return the secret model; EncryptedValue is present, plain value is not.
	return newSecret, nil
}

// GetSecretByID retrieves a secret and its decrypted value.
func (s *secretService) GetSecretByID(ctx context.Context, userID, vaultID, secretID string) (*models.Secret, string, error) {
	if s.secretRepo == nil || s.encryptionService == nil || s.auditService == nil {
		return nil, "", errors.New("secretService: component not initialized")
	}

	_, err := s.checkVaultAccess(ctx, userID, vaultID, "read")
	if err != nil {
		return nil, "", err
	}

	secret, err := s.secretRepo.GetByID(ctx, vaultID, secretID)
	if err != nil {
		return nil, "", fmt.Errorf("failed to get secret '%s' from repository: %w", secretID, err)
	}
	if secret == nil {
		return nil, "", ErrSecretNotFound
	}

	decryptedValue, err := s.encryptionService.Decrypt(secret.EncryptedValue, s.encryptionKey)
	if err != nil {
		// Potentially critical: if decryption fails, it could be data corruption or wrong key.
		// Consider more detailed logging or error handling here.
		return nil, "", fmt.Errorf("%w for secret '%s': %v", ErrDecryptionFailed, secretID, err)
	}

	// Audit Log
	auditLogEntry := models.AuditLog{
		UserID:     userID,
		Action:     "SECRET_ACCESS",
		TargetType: "SECRET",
		TargetID:   secret.ID,
		Details: map[string]interface{}{
			"vault_id":    vaultID,
			"secret_name": secret.Name,
		},
		Timestamp: time.Now().UTC(),
	}
	if auditErr := s.auditService.CreateAuditLog(ctx, auditLogEntry); auditErr != nil {
		fmt.Printf("Warning: failed to create audit log for SECRET_ACCESS (secretID: %s): %v\n", secret.ID, auditErr)
	}
	secret.VaultID = vaultID // For returning context
	return secret, decryptedValue, nil
}

// ListSecrets retrieves all secrets within a vault (does not include decrypted values).
func (s *secretService) ListSecrets(ctx context.Context, userID, vaultID string, paginationParams map[string]string) ([]*models.Secret, error) {
	if s.secretRepo == nil {
		return nil, errors.New("secretService: secretRepo not initialized")
	}

	_, err := s.checkVaultAccess(ctx, userID, vaultID, "read")
	if err != nil {
		return nil, err
	}

	secrets, err := s.secretRepo.GetByVaultID(ctx, vaultID, paginationParams)
	if err != nil {
		return nil, fmt.Errorf("failed to list secrets for vault '%s': %w", vaultID, err)
	}

	// Add VaultID to each secret for context before returning
	for _, secret := range secrets {
		secret.VaultID = vaultID
	}
	return secrets, nil
}

// UpdateSecret updates an existing secret's details and/or value.
func (s *secretService) UpdateSecret(ctx context.Context, userID, vaultID, secretID string, req models.UpdateSecretRequest) (*models.Secret, error) {
	if s.secretRepo == nil || s.encryptionService == nil || s.auditService == nil {
		return nil, errors.New("secretService: component not initialized")
	}

	_, err := s.checkVaultAccess(ctx, userID, vaultID, "write")
	if err != nil {
		return nil, err
	}

	existingSecret, err := s.secretRepo.GetByID(ctx, vaultID, secretID)
	if err != nil {
		return nil, fmt.Errorf("failed to get secret '%s' for update: %w", secretID, err)
	}
	if existingSecret == nil {
		return nil, ErrSecretNotFound
	}

	updatedFields := make(map[string]interface{})
	if req.Name != nil {
		existingSecret.Name = *req.Name
		updatedFields["name"] = *req.Name
	}
	if req.Type != nil {
		existingSecret.Type = *req.Type
		updatedFields["type"] = *req.Type
	}
	if req.Value != nil {
		encryptedValue, err := s.encryptionService.Encrypt(*req.Value, s.encryptionKey)
		if err != nil {
			return nil, fmt.Errorf("%w: %v", ErrEncryptionFailed, err)
		}
		existingSecret.EncryptedValue = encryptedValue
		updatedFields["value_updated"] = true // Don't log the value itself
	}
	// Handle ExpiresAt update if it's part of UpdateSecretRequest and logic is defined.
	// For example:
	// if req.ExpiresAt != nil { // Assuming ExpiresAt is *time.Time in request
	//     existingSecret.ExpiresAt = req.ExpiresAt
	//     updatedFields["expires_at"] = req.ExpiresAt
	// } else if req.Value != nil && existingSecret.ExpiresAt != nil {
	//     // Policy: if value changes, should expiry be cleared or kept?
	//     // If it should be cleared unless explicitly provided:
	//     // existingSecret.ExpiresAt = nil
	//     // updatedFields["expires_at_cleared"] = true
	// }


	existingSecret.UpdatedAt = time.Now().UTC()
	updatedFields["updated_at"] = existingSecret.UpdatedAt


	if err := s.secretRepo.Update(ctx, vaultID, existingSecret); err != nil {
		return nil, fmt.Errorf("failed to update secret '%s': %w", secretID, err)
	}

	// Audit Log
	auditLogEntry := models.AuditLog{
		UserID:     userID,
		Action:     "SECRET_UPDATE",
		TargetType: "SECRET",
		TargetID:   existingSecret.ID,
		Details: map[string]interface{}{
			"vault_id":      vaultID,
			"updatedFields": updatedFields, // Log which fields were changed
		},
		Timestamp: time.Now().UTC(),
	}
	if auditErr := s.auditService.CreateAuditLog(ctx, auditLogEntry); auditErr != nil {
		fmt.Printf("Warning: failed to create audit log for SECRET_UPDATE (secretID: %s): %v\n", existingSecret.ID, auditErr)
	}
	existingSecret.VaultID = vaultID // For returning context
	return existingSecret, nil
}

// DeleteSecret deletes a secret from a vault.
func (s *secretService) DeleteSecret(ctx context.Context, userID, vaultID, secretID string) error {
	if s.secretRepo == nil || s.auditService == nil {
		return errors.New("secretService: component not initialized")
	}

	_, err := s.checkVaultAccess(ctx, userID, vaultID, "write")
	if err != nil {
		return err
	}

	// Optional: Retrieve secret first to log its name, or ensure Delete returns enough info / handles not found gracefully.
	// For now, assume Delete will error if secret not found by repo.
	// existingSecret, err := s.secretRepo.GetByID(ctx, vaultID, secretID)
	// if err != nil || existingSecret == nil {
	//     return ErrSecretNotFound // Or wrap error
	// }

	if err := s.secretRepo.Delete(ctx, vaultID, secretID); err != nil {
		return fmt.Errorf("failed to delete secret '%s': %w", secretID, err)
	}

	// Audit Log
	auditLogEntry := models.AuditLog{
		UserID:     userID,
		Action:     "SECRET_DELETE",
		TargetType: "SECRET",
		TargetID:   secretID,
		Details: map[string]interface{}{
			"vault_id": vaultID,
			// "deleted_secret_name": existingSecret.Name, // If fetched before delete
		},
		Timestamp: time.Now().UTC(),
	}
	if auditErr := s.auditService.CreateAuditLog(ctx, auditLogEntry); auditErr != nil {
		fmt.Printf("Warning: failed to create audit log for SECRET_DELETE (secretID: %s): %v\n", secretID, auditErr)
	}

	return nil
}
