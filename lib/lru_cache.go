package lib

import (
	"sync"
	"time"
)

// LRUCacheEntry represents a single cache entry with value and access tracking
type LRUCacheEntry[V any] struct {
	// value is the cached data
	value V
	// lastAccess tracks when this entry was last accessed for LRU eviction
	lastAccess time.Time
}

// LRUCache is a generic Least Recently Used cache with size limit
type LRUCache[V any] struct {
	// mutex protects concurrent access to the cache
	mutex sync.RWMutex
	// maxSize is the maximum number of entries the cache can hold
	maxSize int
	// entries maps keys to cache entries for O(1) lookup
	entries map[string]*LRUCacheEntry[V]
}

// NewLRUCache creates a new LRU cache with the specified maximum size
func NewLRUCache[V any](maxSize int) *LRUCache[V] {
	return &LRUCache[V]{
		maxSize: maxSize,
		entries: make(map[string]*LRUCacheEntry[V]),
	}
}

// Get retrieves a value from the cache and marks it as recently used
func (c *LRUCache[V]) Get(key string) (V, bool) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	// check if entry exists
	entry, exists := c.entries[key]
	if !exists {
		var zero V
		return zero, false
	}
	// update last access time
	entry.lastAccess = time.Now()
	return entry.value, true
}

// Put adds or updates a value in the cache
func (c *LRUCache[V]) Put(key string, value V) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	now := time.Now()
	// check if entry already exists
	if entry, exists := c.entries[key]; exists {
		// update existing entry
		entry.value = value
		entry.lastAccess = now
		return
	}
	// check if cache is at capacity
	if len(c.entries) >= c.maxSize {
		// find and remove least recently used entry
		c.evictLRUUnsafe()
	}
	// create and add new entry
	c.entries[key] = &LRUCacheEntry[V]{
		value:      value,
		lastAccess: now,
	}
}

// Remove deletes a specific key from the cache
func (c *LRUCache[V]) Remove(key string) bool {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	// check if entry exists
	if _, exists := c.entries[key]; !exists {
		return false
	}
	// remove entry from cache
	delete(c.entries, key)
	return true
}

// Clear removes all entries from the cache
func (c *LRUCache[V]) Clear() {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	// clear all entries
	c.entries = make(map[string]*LRUCacheEntry[V])
}

// Size returns the current number of entries in the cache
func (c *LRUCache[V]) Size() int {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	return len(c.entries)
}

// evictLRUUnsafe removes the least recently used entry (not thread-safe)
func (c *LRUCache[V]) evictLRUUnsafe() {
	var oldestKey string
	var oldestTime time.Time
	first := true
	// find the entry with the oldest last access time
	for key, entry := range c.entries {
		if first || entry.lastAccess.Before(oldestTime) {
			oldestKey = key
			oldestTime = entry.lastAccess
			first = false
		}
	}
	// remove the oldest entry
	if oldestKey != "" {
		delete(c.entries, oldestKey)
	}
}