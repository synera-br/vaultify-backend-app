package db

import (
	"context"
	"fmt"
	"log"
	"strconv" // For parsing pagination params

	"cloud.google.com/go/firestore"
	"google.golang.org/api/iterator"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"vaultify-backend-go/internal/models"
)

const vaultsCollection = "vaults"

// firestoreVaultRepository implements the VaultRepository interface using Firestore.
type firestoreVaultRepository struct {
	client *firestore.Client
}

// NewFirestoreVaultRepository creates a new instance of firestoreVaultRepository.
func NewFirestoreVaultRepository(client *firestore.Client) VaultRepository {
	if client == nil {
		log.Fatal("Firestore client is not initialized for VaultRepository.")
	}
	return &firestoreVaultRepository{client: client}
}

// Create adds a new vault document to Firestore with an auto-generated ID.
// It sets the vault.ID with the new document ID before creation.
// CreatedAt and UpdatedAt fields in models.Vault are handled by serverTimestamp.
func (r *firestoreVaultRepository) Create(ctx context.Context, vault *models.Vault) (string, error) {
	docRef := r.client.Collection(vaultsCollection).NewDoc()
	vault.ID = docRef.ID // Set the ID in the model before saving

	// CreatedAt and UpdatedAt are handled by serverTimestamp tags in the model
	_, err := docRef.Create(ctx, vault)
	if err != nil {
		return "", fmt.Errorf("failed to create vault: %w", err)
	}
	return docRef.ID, nil
}

// GetByID retrieves a vault document from Firestore by its ID.
func (r *firestoreVaultRepository) GetByID(ctx context.Context, vaultID string) (*models.Vault, error) {
	if vaultID == "" {
		return nil, errors.New("vaultID cannot be empty for GetByID operation")
	}
	docSnap, err := r.client.Collection(vaultsCollection).Doc(vaultID).Get(ctx)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return nil, fmt.Errorf("vault with ID '%s' not found: %w", vaultID, ErrNotFound)
		}
		return nil, fmt.Errorf("failed to get vault with ID '%s': %w", vaultID, err)
	}

	var vault models.Vault
	if err := docSnap.DataTo(&vault); err != nil {
		return nil, fmt.Errorf("failed to decode vault data for ID '%s': %w", vaultID, err)
	}
	vault.ID = docSnap.Ref.ID // Ensure ID is populated

	return &vault, nil
}

// GetByOwnerID retrieves all vaults owned by a specific user.
// Pagination is basic: supports "limit" and "startAfter" (document ID).
func (r *firestoreVaultRepository) GetByOwnerID(ctx context.Context, ownerID string, paginationParams map[string]string) ([]*models.Vault, error) {
	if ownerID == "" {
		return nil, errors.New("ownerID cannot be empty for GetByOwnerID operation")
	}

	query := r.client.Collection(vaultsCollection).Where("ownerId", "==", ownerID).OrderBy("createdAt", firestore.Desc) // Default order

	if limitStr, ok := paginationParams["limit"]; ok {
		if limit, err := strconv.Atoi(limitStr); err == nil && limit > 0 {
			query = query.Limit(limit)
		}
	}
	if startAfterDocID, ok := paginationParams["startAfter"]; ok && startAfterDocID != "" {
		// To use StartAfter, we need a DocumentSnapshot of the document to start after.
		// This typically means fetching that document first.
		// For simplicity here, if a 'startAfter' (doc ID) is provided, we fetch that doc.
		// A more robust pagination would use cursor values from the last doc of previous page.
		startAfterSnap, err := r.client.Collection(vaultsCollection).Doc(startAfterDocID).Get(ctx)
		if err == nil {
			query = query.StartAfter(startAfterSnap)
		} else {
			log.Printf("Warning: Could not fetch startAfter document '%s': %v. Pagination may be affected.", startAfterDocID, err)
		}
	}


	iter := query.Documents(ctx)
	defer iter.Stop()

	var vaults []*models.Vault
	for {
		doc, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("failed to iterate vaults for owner '%s': %w", ownerID, err)
		}

		var vault models.Vault
		if err := doc.DataTo(&vault); err != nil {
			// Log and skip problematic document, or return error
			log.Printf("Error decoding vault data (ID: %s) for owner '%s': %v. Skipping.", doc.Ref.ID, ownerID, err)
			continue
		}
		vault.ID = doc.Ref.ID
		vaults = append(vaults, &vault)
	}

	return vaults, nil
}

// Update modifies an existing vault document in Firestore.
// It uses Set with MergeAll to allow partial updates if the service sends partial models.
// UpdatedAt field in models.Vault is handled by serverTimestamp.
func (r *firestoreVaultRepository) Update(ctx context.Context, vault *models.Vault) error {
	if vault.ID == "" {
		return errors.New("vault ID cannot be empty for Update operation")
	}
	// UpdatedAt is handled by serverTimestamp tag in the model
	_, err := r.client.Collection(vaultsCollection).Doc(vault.ID).Set(ctx, vault, firestore.MergeAll)
	if err != nil {
		return fmt.Errorf("failed to update vault with ID '%s': %w", vault.ID, err)
	}
	return nil
}

// Delete removes a vault document from Firestore.
// This does not automatically delete subcollections (e.g., secrets within the vault).
// The service layer is responsible for deleting associated subcollection data.
func (r *firestoreVaultRepository) Delete(ctx context.Context, vaultID string) error {
	if vaultID == "" {
		return errors.New("vaultID cannot be empty for Delete operation")
	}
	_, err := r.client.Collection(vaultsCollection).Doc(vaultID).Delete(ctx)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return fmt.Errorf("vault with ID '%s' not found for deletion: %w", vaultID, ErrNotFound)
		}
		return fmt.Errorf("failed to delete vault with ID '%s': %w", vaultID, err)
	}
	return nil
}

// CountByOwnerID counts the number of vaults owned by a specific user.
// Note: Firestore's count() via GetAggregation is more efficient for large datasets if available and suitable.
// This implementation fetches document snapshots and counts them, which is less efficient for very large counts.
// For typical user vault counts, this is acceptable.
func (r *firestoreVaultRepository) CountByOwnerID(ctx context.Context, ownerID string) (int, error) {
	if ownerID == "" {
		return 0, errors.New("ownerID cannot be empty for CountByOwnerID operation")
	}
	query := r.client.Collection(vaultsCollection).Where("ownerId", "==", ownerID)

	// Using GetAll() to count documents. This is suitable for smaller counts.
	// For very large number of documents, consider using Firestore's aggregation queries (count).
	// Example with aggregation (requires firestore client v1.9.0+):
	// aggQuery := query.NewAggregationQuery().WithCount("all")
	// results, err := aggQuery.Get(ctx)
	// if err != nil {
	//     return 0, fmt.Errorf("failed to count vaults for owner '%s' with aggregation: %w", ownerID, err)
	// }
	// count, ok := results["all"]
	// if !ok {
	//     return 0, fmt.Errorf("aggregation count 'all' not found in results for owner '%s'", ownerID)
	// }
	// if cv, ok := count.(*firestore.AggregationCount); ok {
	//     return int(cv.Value), nil
	// }
	// return 0, fmt.Errorf("unexpected type for aggregation count for owner '%s'", ownerID)

	iter := query.Documents(ctx)
	defer iter.Stop()

	count := 0
	for {
		_, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return 0, fmt.Errorf("failed to iterate vaults for counting (owner '%s'): %w", ownerID, err)
		}
		count++
	}
	return count, nil
}
