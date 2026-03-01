package relayprefs

import (
	"sync"
	"time"
)

// cacheEntry holds a cached relay preference with its expiration time.
type cacheEntry struct {
	prefs     *RelayPrefs
	expiresAt time.Time
}

// cache is a thread-safe in-memory cache for relay preferences.
type cache struct {
	mu      sync.RWMutex
	entries map[string]cacheEntry
	ttl     time.Duration
}

// newCache creates a new cache with the specified TTL.
func newCache(ttl time.Duration) *cache {
	return &cache{
		entries: make(map[string]cacheEntry),
		ttl:     ttl,
	}
}

// Get retrieves cached preferences for a pubkey if they exist and haven't expired.
func (c *cache) Get(pubkey string) (*RelayPrefs, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	entry, ok := c.entries[pubkey]
	if !ok {
		return nil, false
	}

	if time.Now().After(entry.expiresAt) {
		// Expired - don't return it (will be cleaned up later)
		return nil, false
	}

	return entry.prefs, true
}

// Set stores preferences in the cache.
func (c *cache) Set(pubkey string, prefs *RelayPrefs) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.entries[pubkey] = cacheEntry{
		prefs:     prefs,
		expiresAt: time.Now().Add(c.ttl),
	}
}

// Invalidate removes a specific pubkey from the cache.
func (c *cache) Invalidate(pubkey string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	delete(c.entries, pubkey)
}

// InvalidateAll clears the entire cache.
func (c *cache) InvalidateAll() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.entries = make(map[string]cacheEntry)
}

// Cleanup removes expired entries from the cache.
// This can be called periodically to prevent memory growth.
func (c *cache) Cleanup() {
	c.mu.Lock()
	defer c.mu.Unlock()

	now := time.Now()
	for pubkey, entry := range c.entries {
		if now.After(entry.expiresAt) {
			delete(c.entries, pubkey)
		}
	}
}

// Size returns the number of entries in the cache (including expired ones).
func (c *cache) Size() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return len(c.entries)
}
