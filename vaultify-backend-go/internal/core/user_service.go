package core

import (
	"context"
	"errors"
	"time"
	"fmt" // For wrapping errors


	"vaultify-backend-go/internal/db"
	"vaultify-backend-go/internal/models"
	// We will need a way to check for specific error types from the db package.
	// For now, we'll assume a generic way to check for "not found".
	// In a real implementation, db.ErrNotFound would be a typed error.
)

// ErrUserNotFound is returned when a user is not found.
var ErrUserNotFound = errors.New("user not found")


// userService implements the UserService interface.
type userService struct {
	userRepo db.UserRepository
}

// NewUserService creates a new UserService instance.
func NewUserService(userRepo db.UserRepository) UserService {
	return &userService{
		userRepo: userRepo,
	}
}

// GetOrCreate retrieves a user by ID. If the user doesn't exist, it creates a new one.
// Returns the user, a boolean indicating if the user was created, and an error if any.
func (s *userService) GetOrCreate(ctx context.Context, userID, email, displayName, photoURL string) (*models.User, bool, error) {
	if s.userRepo == nil {
		return nil, false, errors.New("UserRepository not initialized in UserService")
	}

	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		if errors.Is(err, db.ErrNotFound) {
			// User not found, create a new one
			newUser := &models.User{
				ID:          userID, // User ID from Firebase Auth is the document ID
				Email:       email,
				DisplayName: displayName,
				PhotoURL:    photoURL,
				Plan:        "FREE",
				CreatedAt:   time.Now().UTC(),
				UpdatedAt:   time.Now().UTC(),
			}
			createErr := s.userRepo.Create(ctx, newUser)
			if createErr != nil {
				return nil, false, fmt.Errorf("failed to create user (id: %s) after not found: %w", userID, createErr)
			}
			return newUser, true, nil // User was created
		}

		// If the error from GetByID was something other than our simulated "not found" error
		return nil, false, fmt.Errorf("failed to get user by ID '%s' from repository: %w", userID, err)
	}

	// err == nil, so user should exist if userRepo.GetByID is well-behaved
	if user == nil {
		// This case implies the repository returned (nil, nil), which is bad practice for GetByID.
		// It should return an error (e.g., db.ErrNotFound) if the item doesn't exist.
		// Log this unexpected behavior from the repository.
		// Depending on strictness, we might even panic or return a different error.
		// For now, we'll treat it as an unexpected state.
		return nil, false, fmt.Errorf("repository returned (nil, nil) for user ID '%s', expected error if not found or user object if found", userID)
	}

	// User exists and err is nil
	// Potentially update user details if they've changed from Auth (e.g., email, displayName, photoURL).
	// This would require an s.userRepo.Update call and careful logic.
	// For this subtask, we keep it simple: if found, return.
	return user, false, nil // User was found, not created


}

// GetByID retrieves a user by their ID.
func (s *userService) GetByID(ctx context.Context, userID string) (*models.User, error) {
	if s.userRepo == nil {
		return nil, errors.New("UserRepository not initialized in UserService")
	}
	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		if errors.Is(err, db.ErrNotFound) {
			// It's important that ErrUserNotFound is defined in this package or a shared core errors package.
			// For now, let's assume it's defined in this (core) package, similar to how VaultService defines its errors.
			// If not, use fmt.Errorf("user with ID '%s' not found: %w", userID, db.ErrNotFound)
			// or return db.ErrNotFound directly if appropriate for the service layer.
			return nil, fmt.Errorf("%w: user with ID '%s'", ErrUserNotFound, userID)
		}
		return nil, fmt.Errorf("failed to get user by ID '%s' from repository: %w", userID, err)
	}
	if user == nil {
		// This case should ideally not be reached if repositories correctly return ErrNotFound.
		// If it is, it indicates an issue with the repository's error handling.
		return nil, fmt.Errorf("%w: user with ID '%s' (repository returned nil user and nil error)", ErrUserNotFound, userID)
	}
	return user, nil
}
