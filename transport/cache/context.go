package cache

import "golang.org/x/net/context"

type key int

var requestInfoKey key

type requestInfo struct {
	useCache bool
	useETag  bool
	debug    bool // if true IsETagEnabled and IsTransportCacheEnabled will return false
}

func NewContext(parent context.Context) context.Context {
	return newContext(parent, &requestInfo{})
}

// IsTransportCacheEnabled returns true if useCache=true and debug=false
func IsTransportCacheEnabled(ctx context.Context) bool {
	if info, ok := fromContext(ctx); ok {
		return info.useCache && !info.debug
	}
	return false
}

// IsETagEnabled returns true if useETag=true and debug=false
func IsETagEnabled(ctx context.Context) bool {
	if info, ok := fromContext(ctx); ok {
		return info.useETag && !info.debug
	}
	return false
}

func IsDebug(ctx context.Context) bool {
	if info, ok := fromContext(ctx); ok {
		return info.debug
	}
	return false
}

func EnableTransportCache(ctx context.Context) {
	if info, ok := fromContext(ctx); ok {
		info.useCache = true
	}
}

// DisableTransportCache disables cache and etag
func DisableTransportCache(ctx context.Context) {
	if info, ok := fromContext(ctx); ok {
		info.useCache = false
		info.useETag = false
	}
}

func EnableETag(ctx context.Context) {
	if info, ok := fromContext(ctx); ok {
		info.useETag = true
	}
}

func DisableETag(ctx context.Context) {
	if info, ok := fromContext(ctx); ok {
		info.useETag = false
	}
}

func EnableDebug(ctx context.Context) {
	if info, ok := fromContext(ctx); ok {
		info.debug = true
	}
}

func DisableDebug(ctx context.Context) {
	if info, ok := fromContext(ctx); ok {
		info.debug = false
	}
}

func NewContextWithTransportCache(parent context.Context) context.Context {
	var info requestInfo
	if c, ok := fromContext(parent); ok {
		info = *c
	}
	info.useCache = true
	return newContext(parent, &info)
}

// NewContextWithoutTransportCache returns new context without using cache and etag
func NewContextWithoutTransportCache(parent context.Context) context.Context {
	var info requestInfo
	if c, ok := fromContext(parent); ok {
		info = *c
	}
	info.useCache = false
	info.useETag = false
	return newContext(parent, &info)
}

func NewContextWithETag(parent context.Context) context.Context {
	var info requestInfo
	if c, ok := fromContext(parent); ok {
		info = *c
	}
	info.useETag = true
	return newContext(parent, &info)
}

func NewContextWithoutETag(parent context.Context) context.Context {
	var info requestInfo
	if c, ok := fromContext(parent); ok {
		info = *c
	}
	info.useETag = false
	return newContext(parent, &info)
}

func NewContextWithDebug(parent context.Context) context.Context {
	var info requestInfo
	if c, ok := fromContext(parent); ok {
		info = *c
	}
	info.debug = true
	return newContext(parent, &info)
}

func NewContextWithoutDebug(parent context.Context) context.Context {
	var info requestInfo
	if c, ok := fromContext(parent); ok {
		info = *c
	}
	info.debug = false
	return newContext(parent, &info)
}

func fromContext(ctx context.Context) (info *requestInfo, ok bool) {
	if val := ctx.Value(requestInfoKey); val != nil {
		info, ok = val.(*requestInfo)
	}
	return
}

func newContext(ctx context.Context, info *requestInfo) context.Context {
	return context.WithValue(ctx, requestInfoKey, info)
}
