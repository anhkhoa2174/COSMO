package cache

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"
)

// MemoryBackend implements an in-memory cache backend.
type MemoryBackend struct {
	mu           sync.RWMutex
	store        map[string]*cacheEntry
	maxSizeBytes int64
	currentSize  int64
}

// cacheEntry represents a cached value with expiration.
type cacheEntry struct {
	value     interface{}
	expiresAt time.Time
	size      int64
}

// NewMemoryBackend creates a new in-memory cache backend.
func NewMemoryBackend(maxSizeBytes int64) *MemoryBackend {
	if maxSizeBytes <= 0 {
		maxSizeBytes = 10 * 1024 * 1024 // Default: 10 MB
	}

	backend := &MemoryBackend{
		store:        make(map[string]*cacheEntry),
		maxSizeBytes: maxSizeBytes,
		currentSize:  0,
	}

	// Start background cleanup goroutine
	go backend.cleanupExpired()

	return backend
}

// Get retrieves a value from the cache.
func (b *MemoryBackend) Get(ctx context.Context, key string) (interface{}, error) {
	b.mu.RLock()
	defer b.mu.RUnlock()

	entry, exists := b.store[key]
	if !exists {
		return nil, nil
	}

	// Check expiration
	if time.Now().After(entry.expiresAt) {
		return nil, nil
	}

	return entry.value, nil
}

// Set stores a value in the cache with TTL.
func (b *MemoryBackend) Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	// Remove old entry if exists
	if oldEntry, exists := b.store[key]; exists {
		b.currentSize -= oldEntry.size
	}

	// Estimate size (simple approximation)
	size := int64(estimateSize(value))

	entry := &cacheEntry{
		value:     value,
		expiresAt: time.Now().Add(ttl),
		size:      size,
	}

	b.store[key] = entry
	b.currentSize += size

	// Evict items if over size limit
	b.evictIfNeeded()

	return nil
}

// Delete removes a key from the cache.
func (b *MemoryBackend) Delete(ctx context.Context, key string) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	if entry, exists := b.store[key]; exists {
		b.currentSize -= entry.size
		delete(b.store, key)
	}

	return nil
}

// DeleteStartsWith removes all keys with the given prefix.
func (b *MemoryBackend) DeleteStartsWith(ctx context.Context, prefix string) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	keysToDelete := make([]string, 0)
	for key := range b.store {
		if strings.HasPrefix(key, prefix) {
			keysToDelete = append(keysToDelete, key)
		}
	}

	for _, key := range keysToDelete {
		if entry, exists := b.store[key]; exists {
			b.currentSize -= entry.size
			delete(b.store, key)
		}
	}

	return nil
}

// Clear removes all keys from the cache.
func (b *MemoryBackend) Clear(ctx context.Context) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.store = make(map[string]*cacheEntry)
	b.currentSize = 0

	return nil
}

// evictIfNeeded evicts oldest entries if cache is over size limit.
func (b *MemoryBackend) evictIfNeeded() {
	for b.currentSize > b.maxSizeBytes && len(b.store) > 0 {
		// Find oldest entry
		var oldestKey string
		var oldestTime time.Time
		first := true

		for key, entry := range b.store {
			if first || entry.expiresAt.Before(oldestTime) {
				oldestKey = key
				oldestTime = entry.expiresAt
				first = false
			}
		}

		// Remove oldest entry
		if oldestKey != "" {
			if entry, exists := b.store[oldestKey]; exists {
				b.currentSize -= entry.size
				delete(b.store, oldestKey)
			}
		}
	}
}

// cleanupExpired removes expired entries periodically.
func (b *MemoryBackend) cleanupExpired() {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		b.mu.Lock()
		now := time.Now()
		for key, entry := range b.store {
			if now.After(entry.expiresAt) {
				b.currentSize -= entry.size
				delete(b.store, key)
			}
		}
		b.mu.Unlock()
	}
}

// estimateSize estimates the size of a value in bytes.
func estimateSize(value interface{}) int {
	// Simple estimation based on type
	switch v := value.(type) {
	case string:
		return len(v)
	case []byte:
		return len(v)
	case int, int8, int16, int32, int64:
		return 8
	case uint, uint8, uint16, uint32, uint64:
		return 8
	case float32, float64:
		return 8
	case bool:
		return 1
	default:
		// Default estimation for complex types
		return 256
	}
}

// Stats returns cache statistics.
func (b *MemoryBackend) Stats() CacheStats {
	b.mu.RLock()
	defer b.mu.RUnlock()

	return CacheStats{
		ItemCount:   len(b.store),
		SizeBytes:   b.currentSize,
		MaxSize:     b.maxSizeBytes,
		Utilization: float64(b.currentSize) / float64(b.maxSizeBytes) * 100,
	}
}

// CacheStats represents cache statistics.
type CacheStats struct {
	ItemCount   int
	SizeBytes   int64
	MaxSize     int64
	Utilization float64 // Percentage
}

// String returns a string representation of cache stats.
func (s CacheStats) String() string {
	return fmt.Sprintf("Items: %d, Size: %d/%d bytes (%.2f%%)",
		s.ItemCount, s.SizeBytes, s.MaxSize, s.Utilization)
}
