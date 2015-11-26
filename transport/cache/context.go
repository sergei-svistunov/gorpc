package cache

import "golang.org/x/net/context"

const requestInfoKey = "http_json_info"

type requestInfo struct {
	UseCache bool
	UseETag  bool
}

func fromContext(ctx context.Context) (info *requestInfo, ok bool) {
	if val := ctx.Value(requestInfoKey); val != nil {
		info, ok = val.(*requestInfo)
	}
	return
}

func NewContext(parent context.Context) context.Context {
	return context.WithValue(parent, requestInfoKey, &requestInfo{})
}

func EnableTrasportCache(ctx context.Context) {
	if info, ok := fromContext(ctx); ok {
		info.UseCache = true
	}
}

func DisableTransportCache(ctx context.Context) {
	if info, ok := fromContext(ctx); ok {
		info.UseCache = false
		info.UseETag = false
	}
}

func IsTransportCacheEnabled(ctx context.Context) bool {
	if info, ok := fromContext(ctx); ok {
		return info.UseCache
	}
	return false
}

func EnableETag(ctx context.Context) {
	if info, ok := fromContext(ctx); ok {
		info.UseETag = true
	}
}

func DisableETag(ctx context.Context) {
	if info, ok := fromContext(ctx); ok {
		info.UseETag = true
	}
}

func IsETagEnabled(ctx context.Context) bool {
	if info, ok := fromContext(ctx); ok {
		return info.UseETag
	}
	return false
}
