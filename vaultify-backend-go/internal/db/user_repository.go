package db

import (
	"context"
	"errors" // For ErrNotFound
	"fmt"
	"log"
	// "time" // Not strictly needed here if serverTimestamp is used for UpdatedAt

	"cloud.google.com/go/firestore"
	// "google.golang.org/api/iterator" // Not used in this version
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"vaultify-backend-go/internal/models"
)

const usersCollection = "users"

// ErrNotFound is a common error for when a document is not found in Firestore.
// It can be defined in a shared errors package (e.g., db/errors.go) if used by multiple repositories.
var ErrNotFound = errors.New("document not found")

// firestoreUserRepository implements the UserRepository interface using Firestore.
type firestoreUserRepository struct {
	client *firestore.Client
}

// NewFirestoreUserRepository creates a new instance of firestoreUserRepository.
func NewFirestoreUserRepository(client *firestore.Client) UserRepository {
	if client == nil {
		log.Fatal("Firestore client is not initialized for UserRepository.")
	}
	return &firestoreUserRepository{client: client}
}

// Create adds a new user document to Firestore.
// The user.ID (Firebase Auth UID) is used as the Firestore document ID.
// CreatedAt and UpdatedAt fields in models.User are expected to be populated by Firestore server-side
// due to the `serverTimestamp` tag.
func (r *firestoreUserRepository) Create(ctx context.Context, user *models.User) error {
	if user.ID == "" {
		return errors.New("user ID cannot be empty for Create operation")
	}
	// The service layer should ensure CreatedAt/UpdatedAt are either set or
	// are zero-value time.Time structs if relying solely on serverTimestamp for the first write.
	// With serverTimestamp, Firestore handles setting these on creation.
	_, err := r.client.Collection(usersCollection).Doc(user.ID).Create(ctx, user)
	if err != nil {
		if status.Code(err) == codes.AlreadyExists {
			return fmt.Errorf("user with ID '%s' already exists: %w", user.ID, err)
		}
		return fmt.Errorf("failed to create user with ID '%s': %w", user.ID, err)
	}
	return nil
}

// GetByID retrieves a user document from Firestore by its ID (Firebase Auth UID).
func (r *firestoreUserRepository) GetByID(ctx context.Context, userID string) (*models.User, error) {
	if userID == "" {
		return nil, errors.New("userID cannot be empty for GetByID operation")
	}
	docSnap, err := r.client.Collection(usersCollection).Doc(userID).Get(ctx)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return nil, fmt.Errorf("user with ID '%s' not found: %w", userID, ErrNotFound)
		}
		return nil, fmt.Errorf("failed to get user with ID '%s': %w", userID, err)
	}

	var user models.User
	if err := docSnap.DataTo(&user); err != nil {
		return nil, fmt.Errorf("failed to decode user data for ID '%s': %w", userID, err)
	}
	user.ID = docSnap.Ref.ID // Ensure ID is populated from the document reference ID

	return &user, nil
}

// Update modifies an existing user document in Firestore.
// It uses Set with MergeAll to only update fields present in the user struct,
// particularly if the service layer sends partial User models.
// If the full User model is sent, it effectively overwrites with the new state.
// The UpdatedAt field in models.User is expected to be populated by Firestore server-side
// due to the `serverTimestamp` tag.
func (r *firestoreUserRepository) Update(ctx context.Context, user *models.User) error {
	if user.ID == "" {
		return errors.New("user ID cannot be empty for Update operation")
	}

	// The models.User struct has `UpdatedAt` tagged with `serverTimestamp`.
	// Firestore should automatically update this field on any successful write operation
	// where the field is part of the struct being set/updated.
	// If specific fields are updated (e.g. using firestore.Update array),
	// then `firestore.ServerTimestamp` might need to be explicitly added for `UpdatedAt`.
	// However, when passing the whole struct `user` to `Set`, the `serverTimestamp` tag
	// on `user.UpdatedAt` should take effect.
	_, err := r.client.Collection(usersCollection).Doc(user.ID).Set(ctx, user, firestore.MergeAll)
	// Using MergeAll is safer for partial updates if the 'user' struct might not contain all fields.
	// If 'user' is always a complete representation of the desired state, MergeAll is not strictly necessary
	// but doesn't harm. If we want to ensure only specific fields are written,
	// we'd construct a map or use []firestore.Update.
	// Given the current service structure, the `user` object passed to Update
	// is typically the one fetched and then modified, so it's complete.
	// If it were a request DTO with optional fields, MergeAll would be more critical.

	if err != nil {
		// Check if the document didn't exist, though Update with MergeAll might create it.
		// To ensure it only updates, the service layer should GetByID first.
		// Firestore's Set with MergeAll will create the document if it doesn't exist.
		// If strict updates (only if exists) are needed, GetByID first or use .Update() method with checks.
		// For now, this behavior is accepted.
		return fmt.Errorf("failed to update user with ID '%s': %w", user.ID, err)
	}
	return nil
}
