package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

// RedisBackend implements a Redis-based cache backend.
type RedisBackend struct {
	client *redis.Client
}

// RedisConfig holds Redis connection configuration.
type RedisConfig struct {
	Addr     string // Redis server address (host:port)
	Password string // Redis password (optional)
	DB       int    // Redis database number
}

// NewRedisBackend creates a new Redis cache backend.
func NewRedisBackend(config RedisConfig) (*RedisBackend, error) {
	client := redis.NewClient(&redis.Options{
		Addr:     config.Addr,
		Password: config.Password,
		DB:       config.DB,
	})

	// Test connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to Redis: %w", err)
	}

	return &RedisBackend{
		client: client,
	}, nil
}

// Get retrieves a value from Redis.
func (b *RedisBackend) Get(ctx context.Context, key string) (interface{}, error) {
	val, err := b.client.Get(ctx, key).Result()
	if err == redis.Nil {
		return nil, nil // Key does not exist
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get key from Redis: %w", err)
	}

	// Try to unmarshal as JSON
	var result interface{}
	if err := json.Unmarshal([]byte(val), &result); err != nil {
		// If JSON unmarshal fails, return as string
		return val, nil
	}

	return result, nil
}

// Set stores a value in Redis with TTL.
func (b *RedisBackend) Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error {
	// Serialize value to JSON
	data, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("failed to marshal value: %w", err)
	}

	if err := b.client.Set(ctx, key, data, ttl).Err(); err != nil {
		return fmt.Errorf("failed to set key in Redis: %w", err)
	}

	return nil
}

// Delete removes a key from Redis.
func (b *RedisBackend) Delete(ctx context.Context, key string) error {
	if err := b.client.Del(ctx, key).Err(); err != nil {
		return fmt.Errorf("failed to delete key from Redis: %w", err)
	}
	return nil
}

// DeleteStartsWith removes all keys with the given prefix from Redis.
func (b *RedisBackend) DeleteStartsWith(ctx context.Context, prefix string) error {
	pattern := prefix + "*"
	iter := b.client.Scan(ctx, 0, pattern, 0).Iterator()

	var keysToDelete []string
	for iter.Next(ctx) {
		keysToDelete = append(keysToDelete, iter.Val())
	}

	if err := iter.Err(); err != nil {
		return fmt.Errorf("failed to scan keys in Redis: %w", err)
	}

	if len(keysToDelete) > 0 {
		if err := b.client.Del(ctx, keysToDelete...).Err(); err != nil {
			return fmt.Errorf("failed to delete keys from Redis: %w", err)
		}
	}

	return nil
}

// Clear removes all keys from the current Redis database.
func (b *RedisBackend) Clear(ctx context.Context) error {
	if err := b.client.FlushDB(ctx).Err(); err != nil {
		return fmt.Errorf("failed to clear Redis database: %w", err)
	}
	return nil
}

// Close closes the Redis connection.
func (b *RedisBackend) Close() error {
	return b.client.Close()
}

// Ping checks the Redis connection.
func (b *RedisBackend) Ping(ctx context.Context) error {
	return b.client.Ping(ctx).Err()
}
