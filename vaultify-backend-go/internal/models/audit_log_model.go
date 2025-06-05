package models

import "time"

// AuditLog represents an audit trail event.
type AuditLog struct {
	ID         string                 `json:"id" firestore:"-"`
	Timestamp  time.Time              `json:"timestamp" firestore:"timestamp,serverTimestamp"`
	UserID     string                 `json:"userId" firestore:"userId"` // Who performed the action
	Action     string                 `json:"action" firestore:"action"` // e.g., "USER_LOGIN", "VAULT_CREATE", "SECRET_ACCESS"
	TargetType string                 `json:"targetType,omitempty" firestore:"targetType,omitempty"` // e.g., "USER", "VAULT", "SECRET"
	TargetID   string                 `json:"targetId,omitempty" firestore:"targetId,omitempty"`   // ID of the affected entity
	IPAddress  string                 `json:"ipAddress,omitempty" firestore:"ipAddress,omitempty"`
	UserAgent  string                 `json:"userAgent,omitempty" firestore:"userAgent,omitempty"`
	Details    map[string]interface{} `json:"details,omitempty" firestore:"details,omitempty"` // Additional information
}
