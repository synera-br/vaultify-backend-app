package core

import (
	"context"
	"fmt"

	"vaultify-backend-go/internal/models"
	"vaultify-backend-go/internal/db" // Corrected import path
)

// auditService implements the AuditService interface.
type auditService struct {
	auditRepo db.AuditRepository // Use the interface from the db package
}

// NewAuditService creates a new AuditService instance.
// It takes an AuditRepository (from the db package) as a dependency.
func NewAuditService(auditRepo db.AuditRepository) AuditService {
	return &auditService{
		auditRepo: auditRepo,
	}
}

// CreateAuditLog creates a new audit log entry.
// It delegates the actual storage to the AuditRepository.
func (s *auditService) CreateAuditLog(ctx context.Context, logEntry models.AuditLog) error {
	if s.auditRepo == nil {
		// This should ideally not happen if the service is constructed properly via NewAuditService
		return fmt.Errorf("AuditRepository not initialized in AuditService")
	}

	// Potentially add more business logic here before or after saving, if needed.
	// For example, validation specific to the core service layer,
	// or emitting an event after successful creation.

	err := s.auditRepo.Create(ctx, logEntry)
	if err != nil {
		// It's good practice to wrap repository errors or handle them specifically
		// if the service layer needs to add more context or perform specific actions based on error types.
		return fmt.Errorf("failed to create audit log via repository: %w", err)
	}

	return nil
}
