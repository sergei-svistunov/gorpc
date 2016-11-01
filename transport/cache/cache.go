package cache

import "time"

type ICache interface {
	// Get returns response (response's content, compressed) by key
	Get(key []byte) *CacheEntry
	// Put puts response in cache
	Put(key []byte, entry *CacheEntry)

	ICacheLocker
}

type TTLAwareCachePutter interface {
	// Put puts response in cache with specified ttl
	PutWithTTL(key []byte, entry *CacheEntry, ttl time.Duration)
}

type ICacheLocker interface {
	Lock(key []byte)
	Unlock(key []byte)
}

type CacheEntry struct {
	Content           []byte
	CompressedContent []byte
	Hash              string
	Body              interface{}
}
