package http_json

import "sync"

// LocalCacheLocker is a helper for cache implementations which prevents multiple requests with the same
// key. If there was a request with the key, then client will be blocked until result is put into the cache.
// It is important to use Unlock with defer statement otherwise cache key can be locked forever.
// It is the responsibility of a cache implementation to unlock cache key after some timeout.
type LocalCacheLocker struct {
	jobs map[string]chan struct{}
	mtx  sync.RWMutex
}

func NewLocalCacheLocker() *LocalCacheLocker {
	return &LocalCacheLocker{
		jobs: make(map[string]chan struct{}),
	}
}

func (l *LocalCacheLocker) Lock(key []byte) {
	jobKey := string(key)
	// optimistic locking. Fast check if there is already job assigned then wait for it.
	l.mtx.RLock()
	job, ok := l.jobs[jobKey]
	l.mtx.RUnlock()
	if ok {
		<-job
		return
	}

	l.mtx.Lock()
	job, ok = l.jobs[jobKey]
	if !ok {
		l.jobs[jobKey] = make(chan struct{})
	}
	l.mtx.Unlock()
	if ok {
		<-job
	}
}

func (l *LocalCacheLocker) Unlock(key []byte) {
	jobKey := string(key)
	// optimistic locking. Fast check if there's no job assigned
	l.mtx.RLock()
	job, ok := l.jobs[jobKey]
	l.mtx.RUnlock()
	if !ok {
		return
	}

	l.mtx.Lock()
	job, ok = l.jobs[jobKey]
	if ok {
		delete(l.jobs, jobKey)
	}
	l.mtx.Unlock()
	if ok {
		close(job)
	}
}
