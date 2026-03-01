package cache

import (
	"context"
	"fmt"
	"time"
)

// Manager manages cache operations with a backend and key maker.
type Manager struct {
	backend  Backend
	keyMaker KeyMaker
}

// NewManager creates a new cache manager.
func NewManager(backend Backend, keyMaker KeyMaker) *Manager {
	if keyMaker == nil {
		keyMaker = NewDefaultKeyMaker()
	}

	return &Manager{
		backend:  backend,
		keyMaker: keyMaker,
	}
}

// Get retrieves a value from the cache.
func (m *Manager) Get(ctx context.Context, key string) (interface{}, error) {
	return m.backend.Get(ctx, key)
}

// Set stores a value in the cache with TTL.
func (m *Manager) Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error {
	return m.backend.Set(ctx, key, value, ttl)
}

// Delete removes a key from the cache.
func (m *Manager) Delete(ctx context.Context, key string) error {
	return m.backend.Delete(ctx, key)
}

// DeleteStartsWith removes all keys with the given prefix.
func (m *Manager) DeleteStartsWith(ctx context.Context, prefix string) error {
	return m.backend.DeleteStartsWith(ctx, prefix)
}

// Clear removes all keys from the cache.
func (m *Manager) Clear(ctx context.Context) error {
	return m.backend.Clear(ctx)
}

// Attempt attempts to get a value from cache or compute it using the provided function.
func (m *Manager) Attempt(ctx context.Context, key string, ttl time.Duration, fn func() (interface{}, error)) (interface{}, error) {
	// Try to get from cache
	cached, err := m.backend.Get(ctx, key)
	if err != nil {
		return nil, fmt.Errorf("cache get error: %w", err)
	}

	if cached != nil {
		return cached, nil
	}

	// Compute value
	value, err := fn()
	if err != nil {
		return nil, err
	}

	// Store in cache (ignore errors on set)
	_ = m.backend.Set(ctx, key, value, ttl)

	return value, nil
}

// MakeKey generates a cache key using the configured key maker.
func (m *Manager) MakeKey(fn interface{}, prefix string, args ...interface{}) string {
	return m.keyMaker.Make(fn, prefix, args...)
}

// CachedFunc represents a function wrapper with caching.
type CachedFunc struct {
	manager *Manager
	prefix  string
	ttl     time.Duration
}

// Cached returns a new cached function wrapper.
func (m *Manager) Cached(prefix string, ttl time.Duration) *CachedFunc {
	return &CachedFunc{
		manager: m,
		prefix:  prefix,
		ttl:     ttl,
	}
}

// Execute executes a function with caching.
func (cf *CachedFunc) Execute(ctx context.Context, fn interface{}, compute func() (interface{}, error), args ...interface{}) (interface{}, error) {
	key := cf.manager.MakeKey(fn, cf.prefix, args...)
	return cf.manager.Attempt(ctx, key, cf.ttl, compute)
}

// GlobalManager is the global cache manager instance.
var GlobalManager *Manager

// SetGlobalManager sets the global cache manager.
func SetGlobalManager(manager *Manager) {
	GlobalManager = manager
}

// GetGlobalManager returns the global cache manager.
func GetGlobalManager() *Manager {
	if GlobalManager == nil {
		// Create default in-memory cache if not set
		GlobalManager = NewManager(NewMemoryBackend(10*1024*1024), NewDefaultKeyMaker())
	}
	return GlobalManager
}
