package cache

import (
	"context"
	"time"
)

// Backend defines the interface for cache storage backends.
type Backend interface {
	// Get retrieves a value from the cache by key.
	Get(ctx context.Context, key string) (interface{}, error)

	// Set stores a value in the cache with a TTL.
	Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error

	// Delete removes a single key from the cache.
	Delete(ctx context.Context, key string) error

	// DeleteStartsWith removes all keys with the given prefix.
	DeleteStartsWith(ctx context.Context, prefix string) error

	// Clear removes all keys from the cache.
	Clear(ctx context.Context) error
}
