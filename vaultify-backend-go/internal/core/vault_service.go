package core

import (
	"context"
	"errors"
	"fmt"
	"time"

	"vaultify-backend-go/internal/db"
	"vaultify-backend-go/internal/models"
	// "vaultify-backend-go/internal/config" // For plan limits from config if needed
)

// Custom errors for the VaultService
var (
	ErrVaultNotFound           = errors.New("vault not found")
	ErrForbiddenAccess         = errors.New("user does not have permission for this action on the vault")
	ErrVaultLimitReached       = errors.New("vault limit reached for the current plan")
	ErrUserNotFound            = errors.New("user not found") // Can be shared or defined per service
	ErrCannotShareWithSelf     = errors.New("cannot share vault with oneself")
	ErrUserAlreadyHasAccess    = errors.New("user already has access with the specified or higher permission")
	ErrShareTargetUserNotFound = errors.New("target user for sharing not found")
	ErrInvalidPermissionLevel  = errors.New("invalid permission level specified for sharing")
	ErrVaultUpdateFailed       = errors.New("failed to update vault")
	ErrVaultDeletionFailed     = errors.New("failed to delete vault")
	ErrSecretDeletionFailed    = errors.New("failed to delete secrets associated with vault")
)

// vaultService implements the VaultService interface.
type vaultService struct {
	vaultRepo    db.VaultRepository
	secretRepo   db.SecretRepository
	userRepo     db.UserRepository
	auditService AuditService
	// config       *config.Config // Uncomment if plan limits are fetched from a config struct
}

// NewVaultService creates a new VaultService instance.
func NewVaultService(
	vr db.VaultRepository,
	sr db.SecretRepository,
	ur db.UserRepository,
	as AuditService,
	// cfg *config.Config, // Uncomment if config is needed
) VaultService {
	return &vaultService{
		vaultRepo:    vr,
		secretRepo:   sr,
		userRepo:     ur,
		auditService: as,
		// config:       cfg, // Uncomment if config is needed
	}
}

// checkVaultLimit is a helper function to check if a user can create more vaults based on their plan.
// This is a conceptual implementation. Actual plan details might come from config or database.
// TODO: Plan limits are currently hardcoded. These should be made configurable
// (e.g., via application config or a dedicated plans management system).
func (s *vaultService) checkVaultLimit(userPlan string, currentVaultCount int) error {
	// These limits should ideally be configurable (e.g., via config.Config)
	// and not hardcoded.
	limits := map[string]int{
		"FREE":       1,    // Example: Free plan allows 1 vault
		"PRO":        10,   // Example: Pro plan allows 10 vaults
		"ENTERPRISE": 1000, // Example: Enterprise plan allows many vaults
	}

	limit, ok := limits[userPlan]
	if !ok {
		// Default to a safe, restrictive limit if the plan is unknown or not configured.
		// This prevents accidental overuse if a new plan is added without updating limits here.
		// Consider logging this event.
		fmt.Printf("Warning: Plan '%s' not found in vault limits configuration, defaulting to 1\n", userPlan)
		limit = 1
	}

	if currentVaultCount >= limit {
		return fmt.Errorf("%w: plan '%s' allows %d vault(s), current count %d", ErrVaultLimitReached, userPlan, limit, currentVaultCount)
	}
	return nil
}


// CreateVault creates a new vault for a user.
// It checks plan limits before creation and records an audit log.
func (s *vaultService) CreateVault(ctx context.Context, userID string, req models.CreateVaultRequest) (*models.Vault, error) {
	if s.userRepo == nil || s.vaultRepo == nil || s.auditService == nil {
		return nil, errors.New("vaultService: component not initialized")
	}

	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user '%s' for plan check: %w", userID, err)
	}
	if user == nil {
		return nil, fmt.Errorf("%w: user with ID '%s' not found for creating vault", ErrUserNotFound, userID)
	}

	currentVaultCount, err := s.vaultRepo.CountByOwnerID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to count vaults for user '%s': %w", userID, err)
	}

	if err := s.checkVaultLimit(user.Plan, currentVaultCount); err != nil {
		return nil, err // ErrVaultLimitReached with details
	}

	newVault := &models.Vault{
		OwnerID:     userID,
		Name:        req.Name,
		Description: req.Description,
		Tags:        req.Tags, // Assumes Tags are directly usable; sanitize if necessary
		SharedWith:  make(map[string]string), // Initialize empty map
		CreatedAt:   time.Now().UTC(),
		UpdatedAt:   time.Now().UTC(),
	}

	vaultID, err := s.vaultRepo.Create(ctx, newVault)
	if err != nil {
		return nil, fmt.Errorf("failed to create vault in repository: %w", err)
	}
	newVault.ID = vaultID // Set the ID returned by the repository

	// Audit Log
	auditLogEntry := models.AuditLog{
		UserID:     userID,
		Action:     "VAULT_CREATE",
		TargetType: "VAULT",
		TargetID:   newVault.ID,
		Timestamp:  time.Now().UTC(), // Firestore serverTimestamp will typically override this
		Details: map[string]interface{}{
			"name":        newVault.Name,
			"description": newVault.Description,
			"tags":        newVault.Tags,
		},
	}
	if auditErr := s.auditService.CreateAuditLog(ctx, auditLogEntry); auditErr != nil {
		// Log the audit error but don't fail the main operation
		fmt.Printf("Warning: failed to create audit log for VAULT_CREATE (vaultID: %s): %v\n", newVault.ID, auditErr)
	}

	return newVault, nil
}

// GetVaultByID retrieves a vault if the user is the owner or has been granted access.
func (s *vaultService) GetVaultByID(ctx context.Context, userID, vaultID string) (*models.Vault, error) {
	if s.vaultRepo == nil {
		return nil, errors.New("vaultService: vaultRepo not initialized")
	}

	vault, err := s.vaultRepo.GetByID(ctx, vaultID)
	if err != nil {
		// Could be a general db error or a true "not found" if repo distinguishes
		// For now, wrap it. If repo returns a specific ErrNotFound, check for it.
		return nil, fmt.Errorf("failed to get vault '%s' from repository: %w", vaultID, err)
	}
	if vault == nil {
		return nil, fmt.Errorf("%w: vault with ID '%s' not found in repository", ErrVaultNotFound, vaultID)
	}

	// Check permissions
	if vault.OwnerID != userID {
		permission, shared := vault.SharedWith[userID]
		if !shared {
			return nil, fmt.Errorf("%w: user '%s' is not owner and not shared with for vault '%s'", ErrForbiddenAccess, userID, vaultID)
		}
		// Could add more granular permission check here if "read" is a specific level
		_ = permission // Use if different permission levels grant read access
	}

	// Audit Log (optional for Get, depends on requirements)
	// Consider if all reads need to be audited. If so:
	/*
	auditLogEntry := models.AuditLog{
		UserID:     userID,
		Action:     "VAULT_ACCESS",
		TargetType: "VAULT",
		TargetID:   vault.ID,
		Timestamp:  time.Now().UTC(),
	}
	if auditErr := s.auditService.CreateAuditLog(ctx, auditLogEntry); auditErr != nil {
		fmt.Printf("Warning: failed to create audit log for VAULT_ACCESS (vaultID: %s): %v\n", vault.ID, auditErr)
	}
	*/

	return vault, nil
}

// ListVaults retrieves vaults owned by the user and shared with the user.
// TODO: Implement fetching shared vaults. Current implementation only fetches owned vaults.
func (s *vaultService) ListVaults(ctx context.Context, userID string, paginationParams map[string]string) ([]*models.Vault, error) {
	if s.vaultRepo == nil {
		return nil, errors.New("vaultService: vaultRepo not initialized")
	}

	ownedVaults, err := s.vaultRepo.GetByOwnerID(ctx, userID, paginationParams)
	if err != nil {
		return nil, fmt.Errorf("failed to list owned vaults for user '%s': %w", userID, err)
	}

	// TODO: Fetch vaults shared with the user.
	// This might require a new repository method like `GetSharedWithUser(ctx, userID, paginationParams)`
	// or expanding `GetByOwnerID` to include an option for shared vaults if the DB schema supports it easily.
	// For now, we only return owned vaults.
	// sharedVaults, err := s.vaultRepo.GetSharedWithUser(ctx, userID, paginationParams)
	// if err != nil {
	//     return nil, fmt.Errorf("failed to list shared vaults for user '%s': %w", userID, err)
	// }
	// combinedVaults := append(ownedVaults, sharedVaults...)
	// return combinedVaults, nil

	return ownedVaults, nil
}

// UpdateVault updates an existing vault if the user is the owner.
func (s *vaultService) UpdateVault(ctx context.Context, userID, vaultID string, req models.UpdateVaultRequest) (*models.Vault, error) {
	if s.vaultRepo == nil || s.auditService == nil {
		return nil, errors.New("vaultService: component not initialized")
	}

	existingVault, err := s.vaultRepo.GetByID(ctx, vaultID)
	if err != nil {
		return nil, fmt.Errorf("failed to get vault '%s' for update: %w", vaultID, err)
	}
	if existingVault == nil {
		return nil, fmt.Errorf("%w: vault with ID '%s' not found for update", ErrVaultNotFound, vaultID)
	}

	if existingVault.OwnerID != userID {
		return nil, fmt.Errorf("%w: user '%s' is not owner of vault '%s'", ErrForbiddenAccess, userID, vaultID)
	}

	// Apply updates
	if req.Name != nil {
		existingVault.Name = *req.Name
	}
	if req.Description != nil {
		existingVault.Description = *req.Description
	}
	if req.Tags != nil { // Assuming replacement of tags, not merging
		existingVault.Tags = *req.Tags
	}
	existingVault.UpdatedAt = time.Now().UTC()

	if err := s.vaultRepo.Update(ctx, existingVault); err != nil {
		return nil, fmt.Errorf("%w: %w", ErrVaultUpdateFailed, err)
	}

	// Audit Log
	auditLogEntry := models.AuditLog{
		UserID:     userID,
		Action:     "VAULT_UPDATE",
		TargetType: "VAULT",
		TargetID:   existingVault.ID,
		Timestamp:  time.Now().UTC(),
		Details: map[string]interface{}{
			"updated_name":        existingVault.Name,
			"updated_description": existingVault.Description,
			"updated_tags":        existingVault.Tags,
		},
	}
	if auditErr := s.auditService.CreateAuditLog(ctx, auditLogEntry); auditErr != nil {
		fmt.Printf("Warning: failed to create audit log for VAULT_UPDATE (vaultID: %s): %v\n", existingVault.ID, auditErr)
	}

	return existingVault, nil
}

// DeleteVault deletes a vault and all its secrets if the user is the owner.
func (s *vaultService) DeleteVault(ctx context.Context, userID, vaultID string) error {
	if s.vaultRepo == nil || s.secretRepo == nil || s.auditService == nil {
		return errors.New("vaultService: component not initialized")
	}

	vault, err := s.vaultRepo.GetByID(ctx, vaultID)
	if err != nil {
		return fmt.Errorf("failed to get vault '%s' for deletion: %w", vaultID, err)
	}
	if vault == nil {
		return fmt.Errorf("%w: vault with ID '%s' not found for deletion", ErrVaultNotFound, vaultID)
	}

	if vault.OwnerID != userID {
		return fmt.Errorf("%w: user '%s' is not owner of vault '%s'", ErrForbiddenAccess, userID, vaultID)
	}

	// Delete all secrets within the vault first
	if err := s.secretRepo.DeleteByVaultID(ctx, vaultID); err != nil {
		// Wrap the error for more context
		return fmt.Errorf("%w: %w (vaultID: %s)", ErrSecretDeletionFailed, err, vaultID)
	}

	// Delete the vault itself
	if err := s.vaultRepo.Delete(ctx, vaultID); err != nil {
		return fmt.Errorf("%w: %w (vaultID: %s)", ErrVaultDeletionFailed, err, vaultID)
	}

	// Audit Log
	auditLogEntry := models.AuditLog{
		UserID:     userID,
		Action:     "VAULT_DELETE",
		TargetType: "VAULT",
		TargetID:   vaultID, // Vault ID is known
		Timestamp:  time.Now().UTC(),
		Details: map[string]interface{}{
			"deleted_vault_name": vault.Name, // Include some details for context
		},
	}
	if auditErr := s.auditService.CreateAuditLog(ctx, auditLogEntry); auditErr != nil {
		fmt.Printf("Warning: failed to create audit log for VAULT_DELETE (vaultID: %s): %v\n", vaultID, auditErr)
	}

	return nil
}

// ShareVault shares a vault with multiple users, setting their permission level.
func (s *vaultService) ShareVault(ctx context.Context, ownerID, vaultID string, req models.ShareVaultRequest) error {
	if s.vaultRepo == nil || s.userRepo == nil || s.auditService == nil {
		return errors.New("vaultService: component not initialized")
	}

	vault, err := s.vaultRepo.GetByID(ctx, vaultID)
	if err != nil {
		return fmt.Errorf("failed to get vault '%s' for sharing: %w", vaultID, err)
	}
	if vault == nil {
		return fmt.Errorf("%w: vault with ID '%s' not found for sharing", ErrVaultNotFound, vaultID)
	}

	if vault.OwnerID != ownerID {
		return fmt.Errorf("%w: user '%s' is not owner of vault '%s', cannot share", ErrForbiddenAccess, ownerID, vaultID)
	}

	// Validate permission level (simple validation for now)
	if req.PermissionLevel != "read" && req.PermissionLevel != "write" {
		return fmt.Errorf("%w: '%s'", ErrInvalidPermissionLevel, req.PermissionLevel)
	}

	if vault.SharedWith == nil {
		vault.SharedWith = make(map[string]string)
	}

	var sharedToUserIDs []string
	for _, targetUserID := range req.UserIDs {
		if targetUserID == ownerID {
			// Log or skip, but don't error out the whole batch for this.
			// Or return ErrCannotShareWithSelf if only one user ID is provided and it's the owner.
			// For batch, better to skip and log.
			fmt.Printf("Warning: User '%s' attempted to share vault '%s' with themselves. Skipping.\n", ownerID, vaultID)
			continue
		}

		// Check if target user exists
		targetUser, err := s.userRepo.GetByID(ctx, targetUserID)
		if err != nil || targetUser == nil {
			// Log or skip. If strict, could return an error.
			fmt.Printf("Warning: Target user '%s' not found for sharing vault '%s'. Skipping. Error: %v\n", targetUserID, vaultID, err)
			continue // Skip this user
		}

		// Check if user already has this or a more permissive access (idempotency)
		// For simplicity, this example overwrites. A more complex logic could check existing permission.
		// if existingPerm, ok := vault.SharedWith[targetUserID]; ok && (existingPerm == req.PermissionLevel || (existingPerm == "write" && req.PermissionLevel == "read")) {
		// 	  fmt.Printf("Info: User '%s' already has '%s' access to vault '%s'. Skipping.\n", targetUserID, existingPerm, vaultID)
		//    continue
		// }


		vault.SharedWith[targetUserID] = req.PermissionLevel
		sharedToUserIDs = append(sharedToUserIDs, targetUserID) // For audit log
	}

	if len(sharedToUserIDs) == 0 && len(req.UserIDs) > 0 {
		// This means all target users were invalid or self.
		// Depending on desired behavior, could return an error or just log that nothing was done.
		// For now, if nothing was effectively shared, we might not need to update the vault.
		if len(req.UserIDs) == 1 && req.UserIDs[0] == ownerID { // Special case for trying to share with self only
			return ErrCannotShareWithSelf
		}
		// If all users were invalid (not found)
		return fmt.Errorf("no valid target users found to share vault '%s' with", vaultID)
	}

	if len(sharedToUserIDs) == 0 { // No actual changes made
		return nil
	}


	vault.UpdatedAt = time.Now().UTC()
	if err := s.vaultRepo.Update(ctx, vault); err != nil {
		return fmt.Errorf("%w for sharing: %w", ErrVaultUpdateFailed, err)
	}

	// Audit Log for each successful share
	for _, sharedUserID := range sharedToUserIDs {
		auditLogEntry := models.AuditLog{
			UserID:     ownerID, // The owner performed the share action
			Action:     "VAULT_SHARE",
			TargetType: "VAULT",
			TargetID:   vaultID,
			Details: map[string]interface{}{
				"shared_with_user_id": sharedUserID,
				"permission_level":    req.PermissionLevel,
			},
			Timestamp: time.Now().UTC(),
		}
		if auditErr := s.auditService.CreateAuditLog(ctx, auditLogEntry); auditErr != nil {
			fmt.Printf("Warning: failed to create audit log for VAULT_SHARE (vaultID: %s, targetUser: %s): %v\n", vaultID, sharedUserID, auditErr)
		}
	}
	return nil
}


// UpdateSharePermissions updates the permission level for a user already shared on a vault.
func (s *vaultService) UpdateSharePermissions(ctx context.Context, ownerID, vaultID, targetUserID, permissionLevel string) error {
	if s.vaultRepo == nil || s.auditService == nil {
		return errors.New("vaultService: component not initialized")
	}

	vault, err := s.vaultRepo.GetByID(ctx, vaultID)
	if err != nil {
		return fmt.Errorf("failed to get vault '%s' for updating permissions: %w", vaultID, err)
	}
	if vault == nil {
		return fmt.Errorf("%w: vault with ID '%s' not found", ErrVaultNotFound, vaultID)
	}

	if vault.OwnerID != ownerID {
		return fmt.Errorf("%w: user '%s' is not owner of vault '%s'", ErrForbiddenAccess, ownerID, vaultID)
	}

	if targetUserID == ownerID {
		return ErrCannotShareWithSelf // Or more specific: "cannot modify owner's implicit permissions"
	}

	if _, ok := vault.SharedWith[targetUserID]; !ok {
		return fmt.Errorf("user '%s' is not currently shared on vault '%s', cannot update permissions", targetUserID, vaultID)
	}

	// Validate permission level
	if permissionLevel != "read" && permissionLevel != "write" {
		return fmt.Errorf("%w: '%s'", ErrInvalidPermissionLevel, permissionLevel)
	}

	vault.SharedWith[targetUserID] = permissionLevel
	vault.UpdatedAt = time.Now().UTC()

	if err := s.vaultRepo.Update(ctx, vault); err != nil {
		return fmt.Errorf("%w for permission update: %w", ErrVaultUpdateFailed, err)
	}

	// Audit Log
	auditLogEntry := models.AuditLog{
		UserID:     ownerID,
		Action:     "VAULT_SHARE_UPDATE_PERMISSION",
		TargetType: "VAULT",
		TargetID:   vaultID,
		Details: map[string]interface{}{
			"target_user_id":      targetUserID,
			"new_permission_level": permissionLevel,
		},
		Timestamp: time.Now().UTC(),
	}
	if auditErr := s.auditService.CreateAuditLog(ctx, auditLogEntry); auditErr != nil {
		fmt.Printf("Warning: failed to create audit log for VAULT_SHARE_UPDATE_PERMISSION (vaultID: %s, targetUser: %s): %v\n", vaultID, targetUserID, auditErr)
	}

	return nil
}

// RemoveShare removes a user's access from a vault.
func (s *vaultService) RemoveShare(ctx context.Context, ownerID, vaultID, targetUserID string) error {
	if s.vaultRepo == nil || s.auditService == nil {
		return errors.New("vaultService: component not initialized")
	}

	vault, err := s.vaultRepo.GetByID(ctx, vaultID)
	if err != nil {
		return fmt.Errorf("failed to get vault '%s' for removing share: %w", vaultID, err)
	}
	if vault == nil {
		return fmt.Errorf("%w: vault with ID '%s' not found", ErrVaultNotFound, vaultID)
	}

	if vault.OwnerID != ownerID {
		return fmt.Errorf("%w: user '%s' is not owner of vault '%s'", ErrForbiddenAccess, ownerID, vaultID)
	}

	if targetUserID == ownerID {
		// This should typically not be allowed as owner has implicit full access.
		return errors.New("cannot remove owner's access from their own vault")
	}

	if _, ok := vault.SharedWith[targetUserID]; !ok {
		return fmt.Errorf("user '%s' is not currently shared on vault '%s', nothing to remove", targetUserID, vaultID)
	}

	delete(vault.SharedWith, targetUserID)
	vault.UpdatedAt = time.Now().UTC()

	if err := s.vaultRepo.Update(ctx, vault); err != nil {
		return fmt.Errorf("%w for removing share: %w", ErrVaultUpdateFailed, err)
	}

	// Audit Log
	auditLogEntry := models.AuditLog{
		UserID:     ownerID,
		Action:     "VAULT_SHARE_REMOVE",
		TargetType: "VAULT",
		TargetID:   vaultID,
		Details: map[string]interface{}{
			"removed_user_id": targetUserID,
		},
		Timestamp: time.Now().UTC(),
	}
	if auditErr := s.auditService.CreateAuditLog(ctx, auditLogEntry); auditErr != nil {
		fmt.Printf("Warning: failed to create audit log for VAULT_SHARE_REMOVE (vaultID: %s, targetUser: %s): %v\n", vaultID, targetUserID, auditErr)
	}

	return nil
}

// Ensure models.CreateVaultRequest, models.UpdateVaultRequest, models.ShareVaultRequest are defined.
// Example (these should be in models package, e.g., models/request_models.go or similar):
/*
package models

// CreateVaultRequest represents the request body for creating a new vault.
type CreateVaultRequest struct {
	Name        string   `json:"name" binding:"required"`
	Description string   `json:"description"`
	Tags        []string `json:"tags"`
}

// UpdateVaultRequest represents the request body for updating an existing vault.
// Pointers are used to distinguish between empty values and fields not provided.
type UpdateVaultRequest struct {
	Name        *string  `json:"name"`
	Description *string  `json:"description"`
	Tags        *[]string `json:"tags"`
}

// ShareVaultRequest represents the request body for sharing a vault.
type ShareVaultRequest struct {
	UserIDs         []string `json:"userIds" binding:"required"` // User IDs to share with
	PermissionLevel string   `json:"permissionLevel" binding:"required"` // e.g., "read", "write"
}
*/
