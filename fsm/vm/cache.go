package vm

import (
	"fmt"
	"sync"

	"github.com/canopy-network/canopy/lib"
)

// InMemoryCache provides a simple in-memory cache for WASM bytecode
type InMemoryCache struct {
	data    map[uint64][]byte
	maxSize uint32
	mu      sync.RWMutex
}

// NewInMemoryCache creates a new in-memory cache with the specified maximum size
func NewInMemoryCache(maxSize uint32) (*InMemoryCache, lib.ErrorI) {
	if maxSize == 0 {
		return nil, ErrCacheMaxSize()
	}

	cache := &InMemoryCache{
		data:    make(map[uint64][]byte),
		maxSize: maxSize,
	}
	return cache, nil
}

// GetCode retrieves WASM bytecode from the cache
func (c *InMemoryCache) GetCode(codeID uint64) ([]byte, lib.ErrorI) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	code, exists := c.data[codeID]
	if !exists {
		return nil, ErrCacheNotFound()
	}

	// Return a copy to prevent modification
	result := make([]byte, len(code))
	copy(result, code)
	return result, nil
}

// SetCode stores WASM bytecode in the cache
func (c *InMemoryCache) SetCode(codeID uint64, code []byte) lib.ErrorI {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Check if we need to evict old entries
	if uint32(len(c.data)) >= c.maxSize {
		c.evictOldest()
	}

	// Store a copy to prevent external modification
	codeCopy := make([]byte, len(code))
	copy(codeCopy, code)
	c.data[codeID] = codeCopy

	return nil
}

// HasCode checks if WASM bytecode exists in the cache
func (c *InMemoryCache) HasCode(codeID uint64) bool {
	c.mu.RLock()
	defer c.mu.RUnlock()

	_, exists := c.data[codeID]
	return exists
}

// RemoveCode removes WASM bytecode from the cache
func (c *InMemoryCache) RemoveCode(codeID uint64) lib.ErrorI {
	c.mu.Lock()
	defer c.mu.Unlock()

	delete(c.data, codeID)
	return nil
}

// Size returns the current number of cached entries
func (c *InMemoryCache) Size() uint32 {
	c.mu.RLock()
	defer c.mu.RUnlock()

	size := uint32(len(c.data))
	return size
}

// Clear removes all entries from the cache
func (c *InMemoryCache) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.data = make(map[uint64][]byte)
}

// evictOldest removes one entry from the cache (simple eviction strategy)
// In a production system, you might want a more sophisticated LRU strategy
func (c *InMemoryCache) evictOldest() {
	if len(c.data) == 0 {
		return
	}

	// Remove an arbitrary entry (maps iterate in random order in Go)
	for codeID := range c.data {
		delete(c.data, codeID)
		break
	}
}

// PersistentCache provides a disk-based cache for WASM bytecode
// This would integrate with the existing BadgerDB storage system
type PersistentCache struct {
	store     lib.RWStoreI
	keyPrefix string
}

// NewPersistentCache creates a new persistent cache using the provided store
func NewPersistentCache(store lib.RWStoreI, keyPrefix string) *PersistentCache {
	cache := &PersistentCache{
		store:     store,
		keyPrefix: keyPrefix,
	}
	return cache
}

// GetCode retrieves WASM bytecode from persistent storage
func (c *PersistentCache) GetCode(codeID uint64) ([]byte, lib.ErrorI) {
	key := c.makeKey(codeID)
	code, err := c.store.Get(key)
	if err != nil {
		return nil, ErrCacheStoreGet(err)
	}
	if len(code) == 0 {
		return nil, ErrCacheNotFound()
	}
	return code, nil
}

// SetCode stores WASM bytecode in persistent storage
func (c *PersistentCache) SetCode(codeID uint64, code []byte) lib.ErrorI {
	key := c.makeKey(codeID)
	if err := c.store.Set(key, code); err != nil {
		return ErrCacheStoreSet(err)
	}
	return nil
}

// HasCode checks if WASM bytecode exists in persistent storage
func (c *PersistentCache) HasCode(codeID uint64) bool {
	key := c.makeKey(codeID)
	_, err := c.store.Get(key)
	exists := err == nil
	return exists
}

// RemoveCode removes WASM bytecode from persistent storage
func (c *PersistentCache) RemoveCode(codeID uint64) lib.ErrorI {
	key := c.makeKey(codeID)
	if err := c.store.Delete(key); err != nil {
		return ErrCacheStoreSet(err)
	}
	return nil
}

// makeKey creates a storage key for the given code ID
func (c *PersistentCache) makeKey(codeID uint64) []byte {
	key := fmt.Sprintf("%s%d", c.keyPrefix, codeID)
	return []byte(key)
}

// HybridCache combines in-memory and persistent caching
type HybridCache struct {
	memory     *InMemoryCache
	persistent *PersistentCache
}

// NewHybridCache creates a cache that uses both memory and persistent storage
func NewHybridCache(memoryMaxSize uint32, store lib.RWStoreI, keyPrefix string) (*HybridCache, lib.ErrorI) {
	memory, err := NewInMemoryCache(memoryMaxSize)
	if err != nil {
		return nil, err
	}

	persistent := NewPersistentCache(store, keyPrefix)

	cache := &HybridCache{
		memory:     memory,
		persistent: persistent,
	}
	return cache, nil
}

// GetCode retrieves code from memory cache first, then persistent cache
func (c *HybridCache) GetCode(codeID uint64) ([]byte, lib.ErrorI) {
	// Try memory cache first
	if code, err := c.memory.GetCode(codeID); err == nil {
		return code, nil
	}

	// Try persistent cache
	code, err := c.persistent.GetCode(codeID)
	if err != nil {
		return nil, err
	}

	// Cache in memory for future access
	c.memory.SetCode(codeID, code)
	return code, nil
}

// SetCode stores code in both memory and persistent cache
func (c *HybridCache) SetCode(codeID uint64, code []byte) lib.ErrorI {
	// Store in persistent cache first
	if err := c.persistent.SetCode(codeID, code); err != nil {
		return err
	}

	// Then cache in memory
	if err := c.memory.SetCode(codeID, code); err != nil {
		return err
	}

	return nil
}

// HasCode checks both memory and persistent cache
func (c *HybridCache) HasCode(codeID uint64) bool {
	memoryHas := c.memory.HasCode(codeID)
	if memoryHas {
		return true
	}
	persistentHas := c.persistent.HasCode(codeID)
	return persistentHas
}

// RemoveCode removes code from both caches
func (c *HybridCache) RemoveCode(codeID uint64) lib.ErrorI {
	// Remove from memory (ignore errors)
	if err := c.memory.RemoveCode(codeID); err != nil {
	}

	// Remove from persistent storage
	if err := c.persistent.RemoveCode(codeID); err != nil {
		return err
	}

	return nil
}
