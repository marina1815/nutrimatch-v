package security

import (
	"crypto/sha256"
	"encoding/base64"
	"sync"
	"time"
)

type TTLCache[T any] struct {
	ttl   time.Duration
	mu    sync.RWMutex
	items map[string]cacheItem[T]
}

type cacheItem[T any] struct {
	value     T
	expiresAt time.Time
}

func NewTTLCache[T any](ttl time.Duration) *TTLCache[T] {
	return &TTLCache[T]{
		ttl:   ttl,
		items: make(map[string]cacheItem[T]),
	}
}

func (c *TTLCache[T]) Get(key string) (T, bool) {
	c.mu.RLock()
	item, ok := c.items[key]
	c.mu.RUnlock()

	var zero T
	if !ok || time.Now().After(item.expiresAt) {
		if ok {
			c.mu.Lock()
			delete(c.items, key)
			c.mu.Unlock()
		}
		return zero, false
	}
	return item.value, true
}

func (c *TTLCache[T]) Set(key string, value T) {
	c.mu.Lock()
	c.items[key] = cacheItem[T]{
		value:     value,
		expiresAt: time.Now().Add(c.ttl),
	}
	c.mu.Unlock()
}

func (c *TTLCache[T]) Delete(key string) {
	c.mu.Lock()
	delete(c.items, key)
	c.mu.Unlock()
}

func SecureCacheKey(parts ...string) string {
	hasher := sha256.New()
	for _, part := range parts {
		hasher.Write([]byte(part))
		hasher.Write([]byte{0})
	}
	return base64.RawURLEncoding.EncodeToString(hasher.Sum(nil))
}
