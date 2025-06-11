package database

import "context"

// FirestoreDB defines the interface for Firestore database operations.
type FirestoreDB interface {
	Get(ctx context.Context, collection string, docID string) (map[string]interface{}, error)
	Add(ctx context.Context, collection string, data interface{}) (string, error)
	Update(ctx context.Context, collection string, docID string, data map[string]interface{}) error
	Delete(ctx context.Context, collection string, docID string) error
	Query(ctx context.Context, collection string, query map[string]interface{}) ([]map[string]interface{}, error)
}
