package http_json

import (
	"github.com/sergei-svistunov/gorpc/transport/cache"
	"sync"
)

type testCache struct {
	values map[string]*cache.CacheEntry
	mtx    sync.RWMutex
}

func (c *testCache) Get(key []byte) *cache.CacheEntry {
	c.mtx.RLock()
	defer c.mtx.RUnlock()

	return c.values[string(key)]
}

func (c *testCache) Put(key []byte, entry *cache.CacheEntry) {
	c.mtx.Lock()
	defer c.mtx.Unlock()

	if c.values == nil {
		c.values = make(map[string]*cache.CacheEntry)
	}

	c.values[string(key)] = entry
}
