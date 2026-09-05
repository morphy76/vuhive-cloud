package cache

import (
	"context"
	"sync"
	"time"

	"github.com/morphy76/vuhive-cloud/internal/bff/application/ports/outbound"
)

var _ outbound.CachePort = (*MemoryCache)(nil)

type cacheItem struct {
	value     []byte
	expiresAt time.Time
}

// MemoryCache provides a thread-safe, in-memory CachePort implementation.
type MemoryCache struct {
	mu    sync.RWMutex
	items map[string]cacheItem
}

// NewMemoryCache constructs an initialized MemoryCache.
func NewMemoryCache() *MemoryCache {
	return &MemoryCache{
		items: make(map[string]cacheItem),
	}
}

// Get retrieves an item from cache, returning false if expired or missing.
func (c *MemoryCache) Get(_ context.Context, key string) ([]byte, bool, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	item, exists := c.items[key]
	if !exists {
		return nil, false, nil
	}

	if time.Now().After(item.expiresAt) {
		return nil, false, nil
	}

	cpy := make([]byte, len(item.value))
	copy(cpy, item.value)
	return cpy, true, nil
}

// Set stores a key-value pair with expiration time.
func (c *MemoryCache) Set(_ context.Context, key string, value []byte, ttl time.Duration) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	cpy := make([]byte, len(value))
	copy(cpy, value)

	c.items[key] = cacheItem{
		value:     cpy,
		expiresAt: time.Now().Add(ttl),
	}
	return nil
}

// Delete removes a key from cache.
func (c *MemoryCache) Delete(_ context.Context, key string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	delete(c.items, key)
	return nil
}
