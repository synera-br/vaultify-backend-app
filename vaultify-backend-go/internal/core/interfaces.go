package core

import (
	"context"

	"vaultify-backend-go/internal/models"
	"vaultify-backend-go/internal/db" // Import for db.AuditRepository
)

// UserService defines the interface for user-related operations.
type UserService interface {
	// GetOrCreate retrieves a user by ID. If the user doesn't exist, it creates a new one with default values.
	GetOrCreate(ctx context.Context, userID, email, displayName, photoURL string) (*models.User, bool, error)
	GetByID(ctx context.Context, userID string) (*models.User, error)
	// Add methods for updating plan details, Stripe customer ID, etc. if needed later.
}

// VaultService defines the interface for vault-related operations.
type VaultService interface {
	CreateVault(ctx context.Context, userID string, req models.CreateVaultRequest) (*models.Vault, error)
	GetVaultByID(ctx context.Context, userID, vaultID string) (*models.Vault, error)
	ListVaults(ctx context.Context, userID string, paginationParams map[string]string) ([]*models.Vault, error)
	UpdateVault(ctx context.Context, userID, vaultID string, req models.UpdateVaultRequest) (*models.Vault, error)
	DeleteVault(ctx context.Context, userID, vaultID string) error
	ShareVault(ctx context.Context, ownerID, vaultID string, req models.ShareVaultRequest) error
	UpdateSharePermissions(ctx context.Context, ownerID, vaultID, targetUserID, permissionLevel string) error
	RemoveShare(ctx context.Context, ownerID, vaultID, targetUserID string) error
	// Add CheckVaultLimit (or similar) if plan limits are checked here
}

// SecretService defines the interface for secret-related operations.
type SecretService interface {
	CreateSecret(ctx context.Context, userID, vaultID string, req models.CreateSecretRequest) (*models.Secret, error)
	GetSecretByID(ctx context.Context, userID, vaultID, secretID string) (*models.Secret, string, error) // Returns Secret and decrypted value
	ListSecrets(ctx context.Context, userID, vaultID string, paginationParams map[string]string) ([]*models.Secret, error) // No encrypted values
	UpdateSecret(ctx context.Context, userID, vaultID, secretID string, req models.UpdateSecretRequest) (*models.Secret, error)
	DeleteSecret(ctx context.Context, userID, vaultID, secretID string) error
	// Add CheckSecretLimit if plan limits apply to secrets
}

// AuditService defines the interface for audit logging operations.
type AuditService interface {
	CreateAuditLog(ctx context.Context, logEntry models.AuditLog) error
}

// EncryptionService defines the interface for cryptographic operations.
type EncryptionService interface {
	Encrypt(plainText string, key []byte) (string, error)
	Decrypt(cipherTextBase64 string, key []byte) (string, error)
}

// UserRepository defines the interface for user data storage operations.
type UserRepository interface {
	// GetByID(ctx context.Context, id string) (*models.User, error)
	// Create(ctx context.Context, user *models.User) error
	// Update(ctx context.Context, user *models.User) error
	// Delete(ctx context.Context, id string) error
}

// VaultRepository defines the interface for vault data storage operations.
type VaultRepository interface {
	// Create(ctx context.Context, vault *models.Vault) error
	// GetByID(ctx context.Context, vaultID string) (*models.Vault, error)
	// ... other methods
}

// SecretRepository defines the interface for secret data storage operations.
type SecretRepository interface {
	// Create(ctx context.Context, secret *models.Secret) error
	// GetByID(ctx context.Context, secretID string, vaultID string) (*models.Secret, error)
	// ... other methods
}

// AuditRepository defines the interface for audit log storage operations.
// This is also defined in internal/db/interfaces.go for separation of concerns,
// but listed here to show what AuditService depends on.
// For actual use by AuditService, it will expect an instance that implements
// the definition from internal/db/interfaces.go, so we use db.AuditRepository here.
// type AuditRepository interface {
// 	Create(ctx context.Context, logEntry models.AuditLog) error
// }
// The AuditService will now expect a db.AuditRepository.
