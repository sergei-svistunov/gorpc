package http_json

import "sync"

type testCache struct {
	values map[string]*CacheEntry
	mtx    sync.RWMutex
}

func (c *testCache) Get(key []byte) *CacheEntry {
	c.mtx.RLock()
	defer c.mtx.RUnlock()

	return c.values[string(key)]
}

func (c *testCache) Put(key []byte, entry *CacheEntry) {
	c.mtx.Lock()
	defer c.mtx.Unlock()

	if c.values == nil {
		c.values = make(map[string]*CacheEntry)
	}

	c.values[string(key)] = entry
}
