package http_json

import (
	"sync"
	"time"
)

// ThrottleCache is a cache implementation which prevents multiple requests with the same
// key during specified timeout. If there was a request with the key, then client will be
// blocked until result is put into the cache or it will be unblocked after specified timeout.
// If throttle timeout is 0 then this cache acts as the normal one.
type ThrottleCache struct {
	cache   ICache
	timeout time.Duration
	jobs    map[string]chan struct{}
	mtx     sync.RWMutex
}

func NewThrottleCache(cache ICache, timeout time.Duration) *ThrottleCache {
	return &ThrottleCache{
		cache:   cache,
		timeout: timeout,
		jobs:    make(map[string]chan struct{}),
	}
}

func (c *ThrottleCache) Get(key []byte) *CacheEntry {
	entry := c.cache.Get(key)
	if c.timeout == 0 || entry != nil {
		return entry
	}

	jobKey := string(key)
	// optimistic locking. Fast check if there is already job assigned then wait for it.
	c.mtx.RLock()
	job, ok := c.jobs[jobKey]
	c.mtx.RUnlock()
	if ok {
		return c.wait(job, key)
	}

	c.mtx.Lock()
	job, ok = c.jobs[jobKey]
	if !ok {
		job = make(chan struct{})
		c.jobs[jobKey] = make(chan struct{})
	}
	c.mtx.Unlock()
	if ok {
		return c.wait(job, key)
	}

	// we assigned the job so we return nil here
	return nil
}

func (c *ThrottleCache) wait(job chan struct{}, key []byte) *CacheEntry {
	select {
	case <-job:
		return c.cache.Get(key)
	case <-time.After(c.timeout):
		return nil
	}
}

func (c *ThrottleCache) Put(key []byte, entry *CacheEntry) {
	c.cache.Put(key, entry)

	if c.timeout == 0 {
		return
	}

	jobKey := string(key)
	// optimistic locking. Fast check if there's no job assigned
	c.mtx.RLock()
	job, ok := c.jobs[jobKey]
	c.mtx.RUnlock()
	if !ok {
		return
	}

	c.mtx.Lock()
	job, ok = c.jobs[jobKey]
	if ok {
		delete(c.jobs, jobKey)
	}
	c.mtx.Unlock()
	if ok {
		close(job)
	}
}
