package http_json

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"errors"
	"net/http"
	"reflect"
	"runtime"
	"strings"
	"time"

	"github.com/sergei-svistunov/gorpc"
	"github.com/sergei-svistunov/gorpc/transport/cache"
	"golang.org/x/net/context"
)

type httpSessionResponse struct {
	Result string      `json:"result"`
	Data   interface{} `json:"data"`
	Error  string      `json:"error"`
}

type APIHandlerCallbacks struct {
	OnInitCtx             func(ctx context.Context, req *http.Request) context.Context
	OnError               func(ctx context.Context, w http.ResponseWriter, req *http.Request, resp interface{}, err *gorpc.CallHandlerError)
	OnPanic               func(ctx context.Context, w http.ResponseWriter, r interface{}, trace []byte, req *http.Request)
	OnStartServing        func(req *http.Request)
	OnEndServing          func(req *http.Request, startTime time.Time)
	OnBeforeWriteResponse func(ctx context.Context, w http.ResponseWriter)
	OnSuccess             func(ctx context.Context, req *http.Request, handlerResponse interface{}, startTime time.Time)
	On404                 func(ctx context.Context, req *http.Request)
	OnCacheHit            func(ctx context.Context, entry *cache.CacheEntry)
	OnCacheMiss           func(ctx context.Context)
	GetCacheKey           func(ctx context.Context, req *http.Request, params interface{}) []byte
}

type APIHandler struct {
	hm        *gorpc.HandlersManager
	cache     cache.ICache
	callbacks APIHandlerCallbacks
}

func NewAPIHandler(hm *gorpc.HandlersManager, cache cache.ICache, callbacks APIHandlerCallbacks) *APIHandler {
	return &APIHandler{
		hm:        hm,
		cache:     cache,
		callbacks: callbacks,
	}
}

func (h *APIHandler) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	startTime := time.Now()

	// TODO: create context with reasonable timeout
	ctx := context.Background()
	if h.callbacks.OnInitCtx != nil {
		ctx = h.callbacks.OnInitCtx(ctx, req)
	}

	if h.callbacks.OnStartServing != nil {
		h.callbacks.OnStartServing(req)
	}

	defer func() {
		if h.callbacks.OnEndServing != nil {
			h.callbacks.OnEndServing(req, startTime)
		}

		if r := recover(); r != nil {
			trace := make([]byte, 16*1024)
			n := runtime.Stack(trace, false)
			trace = trace[:n]

			if h.callbacks.OnPanic != nil {
				h.callbacks.OnPanic(ctx, w, r, trace, req)
			}
			h.writeError(ctx, w, "", http.StatusInternalServerError)
		}
	}()

	if req.Method != "POST" && req.Method != "GET" {
		if h.callbacks.OnError != nil {
			err := &gorpc.CallHandlerError{
				Type: gorpc.ErrorInvalidMethod,
				Err:  errors.New("Invalid method"),
			}
			h.callbacks.OnError(ctx, w, req, nil, err)
		}
		h.writeError(ctx, w, "", http.StatusMethodNotAllowed)
		return
	}

	var resp httpSessionResponse

	handler, params, err := h.parseRequest(ctx, req)
	if err != nil {
		if h.callbacks.OnError != nil {
			h.callbacks.OnError(ctx, w, req, resp, err)
		}
		h.writeError(ctx, w, err.Error(), http.StatusBadRequest)
		return
	}
	if handler == nil {
		if h.callbacks.On404 != nil {
			h.callbacks.On404(ctx, req)
		}
		h.writeError(ctx, w, "", http.StatusNotFound)
		return
	}

	cacheEntry, err := h.callHandlerWithCache(ctx, &resp, req, handler, params)
	if err != nil {
		if h.callbacks.OnError != nil {
			h.callbacks.OnError(ctx, w, req, nil, err)
		}
		switch err.Type {
		case gorpc.ErrorReturnedFromCall:
			resp.Result = "ERROR"
			resp.Data = err.UserMessage()
			resp.Error = err.ErrorCode()
		case gorpc.ErrorInParameters:
			h.writeError(ctx, w, err.UserMessage(), http.StatusBadRequest)
			return
		default:
			h.writeError(ctx, w, "", http.StatusInternalServerError)
			return
		}
	}
	h.writeResponse(ctx, cacheEntry, &resp, w, req, startTime)
}

func (h *APIHandler) CanServe(req *http.Request) bool {
	path := req.URL.Path
	handler := h.hm.FindHandlerByRoute(path)
	return handler != nil
}

func (h *APIHandler) parseRequest(ctx context.Context, req *http.Request) (gorpc.HandlerVersion, reflect.Value, *gorpc.CallHandlerError) {
	if err := req.ParseForm(); err != nil {
		return nil, reflect.ValueOf(nil), &gorpc.CallHandlerError{
			Type: gorpc.ErrorInParameters,
			Err:  err,
		}
	}

	handler := h.hm.FindHandlerByRoute(req.URL.Path)
	if handler == nil {
		return nil, reflect.ValueOf(nil), nil
	}

	jsonRequest := strings.HasPrefix(req.Header.Get("Content-Type"), "application/json")
	if jsonRequest && req.Method != "POST" {
		return nil, reflect.ValueOf(nil), &gorpc.CallHandlerError{
			Type: gorpc.ErrorInvalidMethod,
			Err:  errors.New(http.StatusText(http.StatusBadRequest)),
		}
	}

	var paramsGetter gorpc.IHandlerParameters
	if jsonRequest {
		paramsGetter = &JsonParametersGetter{Req: req.Body}
	} else {
		paramsGetter = &ParametersGetter{Req: req}
	}

	params, err := h.hm.UnmarshalParameters(ctx, handler, paramsGetter)
	if err != nil {
		return nil, reflect.ValueOf(nil), &gorpc.CallHandlerError{
			Type: gorpc.ErrorInParameters,
			Err:  err,
		}
	}
	return handler, params, nil
}

func (h *APIHandler) callHandlerWithCache(ctx context.Context, resp *httpSessionResponse, req *http.Request, handler gorpc.HandlerVersion, params reflect.Value) (cacheEntry *cache.CacheEntry, err *gorpc.CallHandlerError) {
	cacheKey := h.getCacheKey(ctx, req, handler, params)
	if cacheKey == nil {
		return h.callHandler(ctx, cacheKey, resp, req, handler, params)
	}

	h.cache.Lock(cacheKey)
	defer h.cache.Unlock(cacheKey)
	cacheEntry = h.cache.Get(cacheKey)
	if cacheEntry != nil {
		if h.callbacks.OnCacheHit != nil {
			h.callbacks.OnCacheHit(ctx, cacheEntry)
		}

		return
	}
	if h.callbacks.OnCacheMiss != nil {
		h.callbacks.OnCacheMiss(ctx)
	}

	cacheEntry, err = h.callHandler(ctx, cacheKey, resp, req, handler, params)
	if err != nil {
		return
	}

	h.cache.Put(cacheKey, cacheEntry)
	return
}

func (h *APIHandler) callHandler(ctx context.Context, cacheKey []byte, resp *httpSessionResponse, req *http.Request, handler gorpc.HandlerVersion, params reflect.Value) (*cache.CacheEntry, *gorpc.CallHandlerError) {
	handlerResponse, err := h.hm.CallHandler(ctx, handler, params)
	if err != nil {
		return nil, err
	}

	resp.Result = "OK"
	resp.Data = handlerResponse
	return h.createCacheEntry(ctx, resp, cacheKey, req)
}

func (h *APIHandler) getCacheKey(ctx context.Context, req *http.Request, handler gorpc.HandlerVersion, params reflect.Value) []byte {
	if h.cache == nil || !handler.UseCache {
		return nil
	}

	if h.callbacks.GetCacheKey != nil {
		return h.callbacks.GetCacheKey(ctx, req, params.Interface())
	}

	cacheKey, err := json.Marshal(params.Interface())
	if err != nil {
		// TODO: call callback.onError?
		// log.Print(err.Error())
		cacheKey = nil
	}
	return cacheKey
}

func (h *APIHandler) createCacheEntry(ctx context.Context, resp *httpSessionResponse, cacheKey []byte, req *http.Request) (*cache.CacheEntry, *gorpc.CallHandlerError) {
	content, err := json.Marshal(resp)
	if err != nil {
		return nil, &gorpc.CallHandlerError{
			Type: gorpc.ErrorWriteResponse,
			Err:  err,
		}
	}
	cacheEntry := cache.CacheEntry{
		Content: content,
	}
	if len(content) > 1024 && (cacheKey != nil || strings.Contains(req.Header.Get("Accept-Encoding"), "gzip")) {
		buf := new(bytes.Buffer)
		gzipWriter := gzip.NewWriter(buf)
		gzipWriter.Write(content)
		gzipWriter.Close()
		cacheEntry.CompressedContent = buf.Bytes()
	}
	return &cacheEntry, nil
}

func (h *APIHandler) writeResponse(ctx context.Context, cacheEntry *cache.CacheEntry, resp *httpSessionResponse,
	w http.ResponseWriter, req *http.Request, startTime time.Time) {

	if h.callbacks.OnBeforeWriteResponse != nil {
		h.callbacks.OnBeforeWriteResponse(ctx, w)
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	var err error
	if cacheEntry.CompressedContent != nil && strings.Contains(req.Header.Get("Accept-Encoding"), "gzip") {
		w.Header().Set("Content-Encoding", "gzip")
		_, err = w.Write(cacheEntry.CompressedContent)
	} else {
		_, err = w.Write(cacheEntry.Content)
	}

	if err != nil && h.callbacks.OnError != nil {
		handlerError := &gorpc.CallHandlerError{
			Type: gorpc.ErrorWriteResponse,
			Err:  err,
		}
		h.callbacks.OnError(ctx, w, req, resp, handlerError)
		return
	}

	if h.callbacks.OnSuccess != nil {
		h.callbacks.OnSuccess(ctx, req, resp, startTime)
	}
}

func (h *APIHandler) writeError(ctx context.Context, w http.ResponseWriter, error string, code int) {
	if h.callbacks.OnBeforeWriteResponse != nil {
		h.callbacks.OnBeforeWriteResponse(ctx, w)
	}

	if error == "" {
		error = http.StatusText(code)
	}
	http.Error(w, error, code)
}
