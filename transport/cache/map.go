package cache

import "sync"

type mapCache struct {
	values map[string]*CacheEntry
	mtx    sync.RWMutex
	*LocalCacheLocker
}

func NewMapCache() *mapCache {
	return &mapCache{
		values:           make(map[string]*CacheEntry),
		LocalCacheLocker: NewLocalCacheLocker(),
	}
}

func (c *mapCache) Get(key []byte) *CacheEntry {
	c.mtx.RLock()
	defer c.mtx.RUnlock()
	return c.values[string(key)]
}

func (c *mapCache) Put(key []byte, entry *CacheEntry) {
	c.mtx.Lock()
	defer c.mtx.Unlock()
	c.values[string(key)] = entry
}
