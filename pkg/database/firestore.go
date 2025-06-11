package database

import (
	"context"
	"log"

	"cloud.google.com/go/firestore"
	"google.golang.org/api/option"
)

// FirestoreService implements the FirestoreDB interface.
type FirestoreService struct {
	client *firestore.Client
	projectID string
}

// NewFirestoreServiceConfig contains options for creating a new FirestoreService.
type NewFirestoreServiceConfig struct {
	ProjectID       string
	CredentialsFile string // Path to the service account key JSON file. If empty, ADC will be used.
}

// NewFirestoreService creates a new instance of FirestoreService.
func NewFirestoreService(ctx context.Context, cfg NewFirestoreServiceConfig) (FirestoreDB, error) {
	var client *firestore.Client
	var err error

	if cfg.CredentialsFile != "" {
		client, err = firestore.NewClient(ctx, cfg.ProjectID, option.WithCredentialsFile(cfg.CredentialsFile))
	} else {
		// Use Application Default Credentials
		client, err = firestore.NewClient(ctx, cfg.ProjectID)
	}

	if err != nil {
		log.Printf("Failed to create Firestore client: %v", err)
		return nil, err
	}

	log.Println("Successfully connected to Firestore")
	return &FirestoreService{client: client, projectID: cfg.ProjectID}, nil
}

// Get retrieves a document from a Firestore collection.
func (s *FirestoreService) Get(ctx context.Context, collection string, docID string) (map[string]interface{}, error) {
	doc, err := s.client.Collection(collection).Doc(docID).Get(ctx)
	if err != nil {
		log.Printf("Error getting document %s from collection %s: %v", docID, collection, err)
		return nil, err
	}
	return doc.Data(), nil
}

// Add adds a new document to a Firestore collection.
// Returns the ID of the newly created document.
func (s *FirestoreService) Add(ctx context.Context, collection string, data interface{}) (string, error) {
	docRef, _, err := s.client.Collection(collection).Add(ctx, data)
	if err != nil {
		log.Printf("Error adding document to collection %s: %v", collection, err)
		return "", err
	}
	return docRef.ID, nil
}

// Update updates an existing document in a Firestore collection.
func (s *FirestoreService) Update(ctx context.Context, collection string, docID string, data map[string]interface{}) error {
	// Firestore's Update method requires a []firestore.Update.
	// For simplicity, this example uses Set with MergeAll, which overwrites fields.
	// For more granular updates, you'd construct a []firestore.Update.
	_, err := s.client.Collection(collection).Doc(docID).Set(ctx, data, firestore.MergeAll)
	if err != nil {
		log.Printf("Error updating document %s in collection %s: %v", docID, collection, err)
		return err
	}
	return nil
}

// Delete removes a document from a Firestore collection.
func (s *FirestoreService) Delete(ctx context.Context, collection string, docID string) error {
	_, err := s.client.Collection(collection).Doc(docID).Delete(ctx)
	if err != nil {
		log.Printf("Error deleting document %s from collection %s: %v", docID, collection, err)
		return err
	}
	return nil
}

// Query executes a query against a Firestore collection.
// This is a simplified query example. Real-world queries can be more complex.
func (s *FirestoreService) Query(ctx context.Context, collection string, queryParams map[string]interface{}) ([]map[string]interface{}, error) {
	// This is a placeholder for query logic.
	// Firestore queries are typically constructed like:
	// q := s.client.Collection(collection).Where("field", "==", value)
	// iter := q.Documents(ctx)
	// For now, this will return an empty slice.
	log.Printf("Query method called for collection %s with params %v. This is a placeholder.", collection, queryParams)
	return make([]map[string]interface{}, 0), nil
}

// Close closes the Firestore client.
func (s *FirestoreService) Close() error {
	if s.client != nil {
		return s.client.Close()
	}
	return nil
}
