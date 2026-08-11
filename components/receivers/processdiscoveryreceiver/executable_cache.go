package processdiscoveryreceiver

import (
	"sync"
	"time"
)

type executableIdentity struct {
	Device       uint64
	Inode        uint64
	Architecture string
	Size         int64
	ModifiedUnix int64
	GNUBuildID   string
	GoBuildID    string
}

type executableCacheEntry struct {
	result    detectionResult
	found     bool
	expiresAt time.Time
}

type executableCache struct {
	mu      sync.Mutex
	entries map[executableIdentity]executableCacheEntry
	ttl     time.Duration
	maxSize int
}

func newExecutableCache(ttl time.Duration, maxSize int) *executableCache {
	return &executableCache{entries: make(map[executableIdentity]executableCacheEntry), ttl: ttl, maxSize: maxSize}
}

func (c *executableCache) get(key executableIdentity, now time.Time) (detectionResult, bool, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	entry, ok := c.entries[key]
	if !ok || now.After(entry.expiresAt) {
		delete(c.entries, key)
		return detectionResult{}, false, false
	}
	return entry.result, entry.found, true
}

func (c *executableCache) put(key executableIdentity, result detectionResult, found bool, now time.Time) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if len(c.entries) >= c.maxSize {
		for existing := range c.entries {
			delete(c.entries, existing)
			break
		}
	}
	c.entries[key] = executableCacheEntry{result: result, found: found, expiresAt: now.Add(c.ttl)}
}
