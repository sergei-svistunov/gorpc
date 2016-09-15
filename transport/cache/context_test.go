package cache_test

import (
	"github.com/sergei-svistunov/gorpc/transport/cache"
	"github.com/stretchr/testify/assert"
	"golang.org/x/net/context"
	"testing"
)

func TestCacheRequestInfo(t *testing.T) {
	ctx := cache.NewContext(context.Background())
	assert.Equal(t, false, cache.IsTransportCacheEnabled(ctx), "Cache should be disabled for empty context")

	cache.EnableTransportCache(ctx)
	assert.Equal(t, true, cache.IsTransportCacheEnabled(ctx), "Cache should be enabled after calling EnableTransportCache")

	cache.DisableTransportCache(ctx)
	assert.Equal(t, false, cache.IsTransportCacheEnabled(ctx), "Cache should be disabled after calling DisableTransportCache")

	cache.EnableTransportCache(ctx)
	assert.Equal(t, true, cache.IsTransportCacheEnabled(ctx), "Cache should be enabled after calling EnableTransportCache")

	ctx2 := cache.NewContextWithoutTransportCache(ctx)
	assert.Equal(t, true, cache.IsTransportCacheEnabled(ctx), "Cache flag should not be modified in parent context")
	assert.Equal(t, false, cache.IsTransportCacheEnabled(ctx2), "Cache should be disabled in new context after calling NewContextWithoutTransportCache")

	cache.EnableTransportCache(ctx2)
	assert.Equal(t, true, cache.IsTransportCacheEnabled(ctx), "Cache flag should not be modified in parent context")
	assert.Equal(t, true, cache.IsTransportCacheEnabled(ctx2), "Cache should be enabled for new context after calling EnableTransportCache for new context")

	cache.EnableETag(ctx2)
	assert.Equal(t, false, cache.IsETagEnabled(ctx), "ETag flag should not be modified in parent context")
	assert.Equal(t, true, cache.IsETagEnabled(ctx2), "ETag should be enabled for new context after calling EnableETag for new context")

	cache.EnableDebug(ctx2)
	assert.Equal(t, true, cache.IsTransportCacheEnabled(ctx), "Cache flag should not be modified in parent context")
	assert.Equal(t, false, cache.IsDebug(ctx), "Debug flag should not be modified in parent context")
	assert.Equal(t, false, cache.IsETagEnabled(ctx2), "ETag should be disabled for debug mode")
	assert.Equal(t, false, cache.IsTransportCacheEnabled(ctx2), "Cache should be disabled for debug mode")
	cache.DisableDebug(ctx2)

	cache.EnableDebug(ctx)
	assert.Equal(t, false, cache.IsDebug(ctx2), "Debug flag should not be modified in child context")
	assert.Equal(t, true, cache.IsDebug(ctx), "Debug flag should be true after calling EnableDebug")
	assert.Equal(t, false, cache.IsTransportCacheEnabled(ctx), "Cache should be disabled for debug mode")
	assert.Equal(t, false, cache.IsETagEnabled(ctx), "ETag should be disabled for debug mode")
	assert.Equal(t, true, cache.IsETagEnabled(ctx2), "ETag flag should not be modified in child context")
	assert.Equal(t, true, cache.IsTransportCacheEnabled(ctx2), "Chache flag should not be modified in child context")
}
