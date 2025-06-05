package models

import "time"

// Secret represents a secret stored within a vault.
type Secret struct {
	ID             string     `json:"id" firestore:"-"` // Document ID, auto-generated
	VaultID        string     `json:"vaultId" firestore:"-"` // Not stored directly, inferred from subcollection
	Name           string     `json:"name" firestore:"name"`
	Type           string     `json:"type" firestore:"type"` // e.g., "secret", "certificate", "key_value"
	EncryptedValue string     `json:"encryptedValue" firestore:"encryptedValue"`
	ExpiresAt      *time.Time `json:"expiresAt,omitempty" firestore:"expiresAt,omitempty"` // Pointer to be optional
	CreatedAt      time.Time  `json:"createdAt" firestore:"createdAt,serverTimestamp"`
	UpdatedAt      time.Time  `json:"updatedAt" firestore:"updatedAt,serverTimestamp"`
}
