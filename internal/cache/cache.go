// Package cache provides a simple thread-safe cache implementation.
package cache

import (
	"sync"
)

// Cache is a simple thread-safe cache with size limits.
type Cache struct {
	mu    sync.RWMutex
	items map[string]interface{}
	max   int
}

// New creates a new cache with the specified maximum size.
func New(maxSize int) *Cache {
	return &Cache{
		items: make(map[string]interface{}),
		max:   maxSize,
	}
}

// Get retrieves a value from the cache.
func (c *Cache) Get(key string) (interface{}, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	value, exists := c.items[key]
	return value, exists
}

// Set stores a value in the cache.
func (c *Cache) Set(key string, value interface{}) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Simple eviction: if at capacity, clear half the cache
	if len(c.items) >= c.max {
		count := 0
		for k := range c.items {
			delete(c.items, k)
			count++
			if count >= c.max/2 {
				break
			}
		}
	}

	c.items[key] = value
}

// Clear removes all items from the cache.
func (c *Cache) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.items = make(map[string]interface{})
}

// Size returns the current number of items in the cache.
func (c *Cache) Size() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return len(c.items)
}
