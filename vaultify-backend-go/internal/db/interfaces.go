package db

import (
	"context"
	"vaultify-backend-go/internal/models"
)

// UserRepository defines the interface for user data storage operations.
type UserRepository interface {
	GetByID(ctx context.Context, userID string) (*models.User, error)
	Create(ctx context.Context, user *models.User) error
	Update(ctx context.Context, user *models.User) error // For future use
	// Delete removes a user from the database by their ID.
	// Delete(ctx context.Context, id string) error
	// GetByEmail retrieves a user by their email.
	// GetByEmail(ctx context.Context, email string) (*models.User, error)
}

// VaultRepository defines the interface for vault data storage operations.
type VaultRepository interface {
	Create(ctx context.Context, vault *models.Vault) (string, error) // Returns new vault ID
	GetByID(ctx context.Context, vaultID string) (*models.Vault, error)
	GetByOwnerID(ctx context.Context, ownerID string, paginationParams map[string]string) ([]*models.Vault, error)
	Update(ctx context.Context, vault *models.Vault) error
	Delete(ctx context.Context, vaultID string) error
	CountByOwnerID(ctx context.Context, ownerID string) (int, error) // For plan limits
	// Add methods for querying shared vaults if necessary, or handle in service layer
}

// SecretRepository defines the interface for secret data storage operations.
type SecretRepository interface {
	Create(ctx context.Context, vaultID string, secret *models.Secret) (string, error) // Returns new secret ID
	GetByID(ctx context.Context, vaultID, secretID string) (*models.Secret, error)
	GetByVaultID(ctx context.Context, vaultID string, paginationParams map[string]string) ([]*models.Secret, error)
	Update(ctx context.Context, vaultID string, secret *models.Secret) error
	Delete(ctx context.Context, vaultID, secretID string) error
	DeleteByVaultID(ctx context.Context, vaultID string) error // Already defined
	CountByVaultID(ctx context.Context, vaultID string) (int, error) // For plan limits if secrets per vault
}

// AuditRepository defines the interface for audit log data storage operations.
type AuditRepository interface {
	Create(ctx context.Context, logEntry models.AuditLog) error
	// GetByUserID retrieves audit logs for a specific user.
	// GetByUserID(ctx context.Context, userID string, limit int, offset int) ([]*models.AuditLog, error)
	// GetByTargetID retrieves audit logs for a specific target entity.
	// GetByTargetID(ctx context.Context, targetType string, targetID string, limit int, offset int) ([]*models.AuditLog, error)
}
