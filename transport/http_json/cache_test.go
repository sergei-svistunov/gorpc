package http_json

import (
	"sync"
	"testing"
	"time"
)

type testCache struct {
	values map[string]*CacheEntry
	mtx    sync.RWMutex
	*LocalCacheLocker
}

func NewTestCache() *testCache {
	return &testCache{
		values:           make(map[string]*CacheEntry),
		LocalCacheLocker: NewLocalCacheLocker(),
	}
}

func (c *testCache) Get(key []byte) *CacheEntry {
	c.mtx.RLock()
	defer c.mtx.RUnlock()
	return c.values[string(key)]
}

func (c *testCache) Put(key []byte, entry *CacheEntry) {
	c.mtx.Lock()
	defer c.mtx.Unlock()
	c.values[string(key)] = entry
}

func TestLockCache(t *testing.T) {
	testKey := []byte("test_key")
	cache := NewTestCache()
	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		defer wg.Done()

		cache.Lock(testKey)
		defer cache.Unlock(testKey)
		_ = cache.Get(testKey)
		time.Sleep(50 * time.Millisecond)
		cache.Put(testKey, &CacheEntry{Content: []byte("test_content")})
	}()

	for i := 0; i < 3; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()

			// sleep to be sure that our first goroutine starts first
			time.Sleep(5 * time.Millisecond)

			cache.Lock(testKey)
			defer cache.Unlock(testKey)
			entry := cache.Get(testKey)
			if entry == nil || string(entry.Content) != "test_content" {
				t.FailNow()
			}
		}()
	}

	wg.Wait()
}
