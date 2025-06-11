package cache

import (
	"context"
	"time"
	"log"

	"github.com/go-redis/redis/v8"
)

// RedisCache is an implementation of the Cache interface using Redis.
type RedisCache struct {
	client *redis.Client
	ctx    context.Context
}

// NewRedisCacheConfig contains options for creating a new RedisCache.
type NewRedisCacheConfig struct {
	Address  string
	Password string
	DB       int
}

// NewRedisCache creates a new RedisCache.
func NewRedisCache(cfg NewRedisCacheConfig) (Cache, error) {
	rdb := redis.NewClient(&redis.Options{
		Addr:     cfg.Address,
		Password: cfg.Password,
		DB:       cfg.DB,
	})

	ctx := context.Background()
	_, err := rdb.Ping(ctx).Result()
	if err != nil {
		log.Printf("Failed to connect to Redis: %v", err)
		return nil, err
	}

	log.Println("Successfully connected to Redis")
	return &RedisCache{client: rdb, ctx: ctx}, nil
}

// Get retrieves a value from Redis.
func (r *RedisCache) Get(key string) (string, error) {
	val, err := r.client.Get(r.ctx, key).Result()
	if err == redis.Nil {
		return "", nil // Key does not exist
	} else if err != nil {
		log.Printf("Error getting key %s from Redis: %v", key, err)
		return "", err
	}
	return val, nil
}

// Set stores a value in Redis.
func (r *RedisCache) Set(key string, value interface{}, expiration time.Duration) error {
	err := r.client.Set(r.ctx, key, value, expiration).Err()
	if err != nil {
		log.Printf("Error setting key %s in Redis: %v", key, err)
		return err
	}
	return nil
}

// Delete removes a value from Redis.
func (r *RedisCache) Delete(key string) error {
	err := r.client.Del(r.ctx, key).Err()
	if err != nil {
		log.Printf("Error deleting key %s from Redis: %v", key, err)
		return err
	}
	return nil
}
