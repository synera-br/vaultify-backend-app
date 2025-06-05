package models

import "time"

// User represents a user in the system.
type User struct {
	ID                   string    `json:"id" firestore:"-"` // Firebase Auth UID, will be the document ID
	Email                string    `json:"email"`
	DisplayName          string    `json:"displayName,omitempty"`
	PhotoURL             string    `json:"photoURL,omitempty"`
	Plan                 string    `json:"plan"` // e.g., "FREE", "PRO", "ENTERPRISE"
	StripeCustomerID     string    `json:"stripeCustomerId,omitempty"`
	StripeSubscriptionID string    `json:"stripeSubscriptionId,omitempty"`
	SubscriptionStatus   string    `json:"subscriptionStatus,omitempty"` // e.g., "active", "canceled"
	CreatedAt            time.Time `json:"createdAt" firestore:"createdAt,serverTimestamp"`
	UpdatedAt            time.Time `json:"updatedAt" firestore:"updatedAt,serverTimestamp"`
}
