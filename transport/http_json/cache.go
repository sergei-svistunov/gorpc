package http_json

type ICache interface {
	// Get returns response (response's content, compressed) by key
	Get(key []byte) *CacheEntry
	// Put puts response in cache
	Put(key []byte, entry *CacheEntry)
}

type CacheEntry struct {
	Content           []byte
	CompressedContent []byte
}
