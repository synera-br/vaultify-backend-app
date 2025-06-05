package models

import "time"

// Vault represents a collection of secrets.
type Vault struct {
	ID          string            `json:"id" firestore:"-"` // Document ID, auto-generated
	OwnerID     string            `json:"ownerId" firestore:"ownerId"` // Firebase Auth UID of the owner
	Name        string            `json:"name" firestore:"name"`
	Description string            `json:"description,omitempty" firestore:"description,omitempty"`
	Tags        []string          `json:"tags,omitempty" firestore:"tags,omitempty"`
	SharedWith  map[string]string `json:"sharedWith,omitempty" firestore:"sharedWith,omitempty"` // map of userID to permissionLevel ("read", "write")
	CreatedAt   time.Time         `json:"createdAt" firestore:"createdAt,serverTimestamp"`
	UpdatedAt   time.Time         `json:"updatedAt" firestore:"updatedAt,serverTimestamp"`
}
