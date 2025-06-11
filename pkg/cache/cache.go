package cache

import "time"

// Cache defines the interface for caching services.
type Cache interface {
	Get(key string) (string, error)
	Set(key string, value interface{}, expiration time.Duration) error
	Delete(key string) error
}
