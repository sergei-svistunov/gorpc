package http_json

import (
	"sync"
	"testing"
	"time"
)

func TestThrottleCache(t *testing.T) {
	cache := NewThrottleCache(&testCache{}, 100*time.Millisecond)
	_ = cache.Get([]byte("test_key"))

	var wg sync.WaitGroup
	for i := 0; i < 3; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()

			entry := cache.Get([]byte("test_key"))
			if string(entry.Content) != "test_content" {
				t.FailNow()
			}
		}()
	}

	cache.Put([]byte("test_key"), &CacheEntry{Content: []byte("test_content")})
	wg.Wait()
}

func TestThrottleCacheTimeout(t *testing.T) {
	cache := NewThrottleCache(&testCache{}, 100*time.Millisecond)
	_ = cache.Get([]byte("test_key"))

	var wg sync.WaitGroup
	for i := 0; i < 3; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()

			entry := cache.Get([]byte("test_key"))
			if entry != nil {
				t.FailNow()
			}
		}()
	}

	time.Sleep(2 * 100 * time.Millisecond)
	cache.Put([]byte("test_key"), &CacheEntry{Content: []byte("test_content")})
	wg.Wait()
}
