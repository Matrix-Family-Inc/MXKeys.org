//go:build integration

package integration

import (
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

type mockCache struct {
	mu     sync.RWMutex
	data   map[string]cacheEntry
	hits   int64
	misses int64
}

type cacheEntry struct {
	value     interface{}
	expiresAt time.Time
}

func newMockCache() *mockCache {
	return &mockCache{
		data: make(map[string]cacheEntry),
	}
}

func (c *mockCache) Get(key string) (interface{}, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	entry, ok := c.data[key]
	if !ok {
		atomic.AddInt64(&c.misses, 1)
		return nil, false
	}

	if time.Now().After(entry.expiresAt) {
		atomic.AddInt64(&c.misses, 1)
		return nil, false
	}

	atomic.AddInt64(&c.hits, 1)
	return entry.value, true
}

func (c *mockCache) Set(key string, value interface{}, ttl time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.data[key] = cacheEntry{
		value:     value,
		expiresAt: time.Now().Add(ttl),
	}
}

func (c *mockCache) Stats() (hits, misses int64) {
	return atomic.LoadInt64(&c.hits), atomic.LoadInt64(&c.misses)
}

func TestCacheHitSecondRequest(t *testing.T) {
	cache := newMockCache()
	serverName := "matrix.org"
	keyData := map[string]string{"key": "testvalue"}

	_, ok := cache.Get(serverName)
	if ok {
		t.Error("first request should be cache miss")
	}

	cache.Set(serverName, keyData, time.Hour)

	val, ok := cache.Get(serverName)
	if !ok {
		t.Fatal("second request should be cache hit")
	}

	data, ok := val.(map[string]string)
	if !ok {
		t.Fatal("cached value has wrong type")
	}

	if data["key"] != "testvalue" {
		t.Errorf("cached value mismatch: got %s", data["key"])
	}

	hits, misses := cache.Stats()
	if hits != 1 {
		t.Errorf("expected 1 hit, got %d", hits)
	}
	if misses != 1 {
		t.Errorf("expected 1 miss, got %d", misses)
	}
}

func TestCacheExpiration(t *testing.T) {
	cache := newMockCache()
	serverName := "expiring.server"

	cache.Set(serverName, "value", 10*time.Millisecond)

	_, ok := cache.Get(serverName)
	if !ok {
		t.Error("should hit before expiration")
	}

	time.Sleep(20 * time.Millisecond)

	_, ok = cache.Get(serverName)
	if ok {
		t.Error("should miss after expiration")
	}
}

func TestCacheIsolation(t *testing.T) {
	cache := newMockCache()

	cache.Set("server1", "value1", time.Hour)
	cache.Set("server2", "value2", time.Hour)

	val1, ok := cache.Get("server1")
	if !ok || val1 != "value1" {
		t.Error("server1 cache mismatch")
	}

	val2, ok := cache.Get("server2")
	if !ok || val2 != "value2" {
		t.Error("server2 cache mismatch")
	}

	_, ok = cache.Get("server3")
	if ok {
		t.Error("server3 should not exist")
	}
}

func TestCacheConcurrency(t *testing.T) {
	cache := newMockCache()
	serverName := "concurrent.server"
	cache.Set(serverName, "initial", time.Hour)

	var wg sync.WaitGroup
	errCh := make(chan string, 100)

	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			val, ok := cache.Get(serverName)
			if !ok {
				return
			}
			if val != "initial" {
				errCh <- "unexpected value"
			}
		}()
	}

	wg.Wait()
	close(errCh)

	for errMsg := range errCh {
		t.Error(errMsg)
	}
}

func TestCacheUpdateOnRefetch(t *testing.T) {
	cache := newMockCache()
	serverName := "updating.server"

	cache.Set(serverName, "old_key", time.Hour)

	val, ok := cache.Get(serverName)
	if !ok || val != "old_key" {
		t.Error("initial cache miss or wrong value")
	}

	cache.Set(serverName, "new_key", time.Hour)

	val, ok = cache.Get(serverName)
	if !ok || val != "new_key" {
		t.Error("updated cache miss or wrong value")
	}
}
