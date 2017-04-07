package http_json

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"reflect"
	"runtime"
	"strings"
	"time"

	"github.com/sergei-svistunov/gorpc"
	"github.com/sergei-svistunov/gorpc/debug"
	"github.com/sergei-svistunov/gorpc/transport/cache"
	"context"
)

var PrintDebug = false

//easyjson:json
type httpSessionResponse struct {
	Result string       `json:"result"`
	Data   interface{}  `json:"data"`
	Error  string       `json:"error"`
	Debug  *debug.Debug `json:"debug,omitempty"`
}

type APIHandlerCallbacks struct {
	OnInitCtx             func(ctx context.Context, req *http.Request) context.Context
	OnError               func(ctx context.Context, w http.ResponseWriter, req *http.Request, resp interface{}, err *gorpc.CallHandlerError)
	OnPanic               func(ctx context.Context, w http.ResponseWriter, r interface{}, trace []byte, req *http.Request)
	OnStartServing        func(ctx context.Context, req *http.Request)
	OnEndServing          func(ctx context.Context, req *http.Request, startTime time.Time)
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
	timeout   time.Duration
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

	var (
		ctx    context.Context
		cancel context.CancelFunc
	)
	if h.timeout > 0 {
		ctx, cancel = context.WithTimeout(context.Background(), h.timeout)
		defer cancel()
	} else {
		ctx = context.Background()
	}

	ctx = cache.NewContext(ctx)
	if h.callbacks.OnInitCtx != nil {
		ctx = h.callbacks.OnInitCtx(ctx, req)
	}

	if h.callbacks.OnStartServing != nil {
		h.callbacks.OnStartServing(ctx, req)
	}

	defer func() {
		if h.callbacks.OnEndServing != nil {
			h.callbacks.OnEndServing(ctx, req, startTime)
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

	done := make(chan bool, 1)
	var cacheEntry *cache.CacheEntry

	go func() {
		defer func() {
			if r := recover(); r != nil {
				trace := make([]byte, 16*1024)
				n := runtime.Stack(trace, false)
				trace = trace[:n]

				if h.callbacks.OnPanic != nil {
					h.callbacks.OnPanic(ctx, w, r, trace, req)
				}
				err = &gorpc.CallHandlerError{
					Type: gorpc.ErrorPanic,
					Err:  fmt.Errorf("Panic in handler:\n%#v\n\n%s", r, string(trace)),
				}
			}
			done <- true
		}()
		cacheEntry, err = h.callHandlerWithCache(ctx, &resp, req, handler, params)
	}()

	// Wait handler or ctx timeout
	select {
	case <-ctx.Done():
	case <-done:
	}

	if ctx.Err() == context.DeadlineExceeded {
		h.writeTimeoutError(ctx, req, w)
		return
	}
	if err != nil {
		if err.Type != gorpc.ErrorPanic && h.callbacks.OnError != nil {
			h.callbacks.OnError(ctx, w, req, nil, err)
		}
		switch err.Type {
		case gorpc.ErrorInParameters:
			h.writeError(ctx, w, err.UserMessage(), http.StatusBadRequest)
		default:
			h.writeInternalError(ctx, w, err.Error())
		}
		return
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
	if cacheKey == nil || cache.IsDebug(ctx) {
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

	if cache.IsTransportCacheEnabled(ctx) {
		if cache.IsETagEnabled(ctx) {
			cacheEntry.Hash, _ = cache.ETagHash(cacheEntry.Content)
		}
		ttl := cache.TTL(ctx)
		if p, ok := h.cache.(cache.TTLAwareCachePutter); ok && ttl > 0 {
			p.PutWithTTL(cacheKey, cacheEntry, ttl)
		} else {
			h.cache.Put(cacheKey, cacheEntry)
		}
	}
	return
}

func (h *APIHandler) callHandler(ctx context.Context, cacheKey []byte, resp *httpSessionResponse, req *http.Request, handler gorpc.HandlerVersion, params reflect.Value) (*cache.CacheEntry, *gorpc.CallHandlerError) {
	if h.IsDebug(req) {
		ctx = context.WithValue(ctx, debug.DebugContextKey, debug.NewDebug())
	}
	handlerResponse, err := h.hm.CallHandler(ctx, handler, params)
	if err != nil {
		if err.Type == gorpc.ErrorReturnedFromCall {
			resp.Result = "ERROR"
			resp.Data = err.UserMessage()
			resp.Error = err.ErrorCode()
			return h.createCacheEntry(ctx, resp, nil, req)
		}
		return nil, err
	}

	resp.Result = "OK"
	resp.Data = handlerResponse
	if debugObj, ok := debug.GetDebugFromContext(ctx); ok {
		resp.Debug = debugObj
	}
	return h.createCacheEntry(ctx, resp, cacheKey, req)
}

func (h *APIHandler) getCacheKey(ctx context.Context, req *http.Request, handler gorpc.HandlerVersion, params reflect.Value) []byte {
	if h.cache == nil {
		return nil
	}

	if h.callbacks.GetCacheKey != nil {
		if cacheKey := h.callbacks.GetCacheKey(ctx, req, params.Interface()); cacheKey != nil {
			return cacheKey
		}
	}

	buf := bytes.NewBufferString(handler.Route)
	encoder := json.NewEncoder(buf)
	err := encoder.Encode(params.Interface())
	if err != nil {
		// TODO: call callback.onError?
		return nil
	}
	return buf.Bytes()
}

func (h *APIHandler) createCacheEntry(ctx context.Context, resp *httpSessionResponse, cacheKey []byte, req *http.Request) (*cache.CacheEntry, *gorpc.CallHandlerError) {
	content, err := resp.MarshalJSON()
	if err != nil {
		return nil, &gorpc.CallHandlerError{
			Type: gorpc.ErrorWriteResponse,
			Err:  err,
		}
	}
	cacheEntry := cache.CacheEntry{
		Content: content,
	}
	if len(content) > 4096 && cacheKey != nil && strings.Contains(req.Header.Get("Accept-Encoding"), "gzip") {
		buf := new(bytes.Buffer)
		if gzipWriter, err := gzip.NewWriterLevel(buf, gzip.BestSpeed); err == nil {
			gzipWriter.Write(content)
			gzipWriter.Close()
			cacheEntry.CompressedContent = buf.Bytes()
		}
	}
	return &cacheEntry, nil
}

func (h *APIHandler) writeResponse(ctx context.Context, cacheEntry *cache.CacheEntry, resp *httpSessionResponse,
	w http.ResponseWriter, req *http.Request, startTime time.Time) {

	if h.callbacks.OnBeforeWriteResponse != nil {
		h.callbacks.OnBeforeWriteResponse(ctx, w)
	}

	if cacheEntry.Hash != "" {
		w.Header().Set("Etag", cacheEntry.Hash)
		if cacheEntry.Hash == req.Header.Get("If-None-Match") {
			w.WriteHeader(http.StatusNotModified)
			if h.callbacks.OnSuccess != nil {
				h.callbacks.OnSuccess(ctx, req, resp, startTime)
			}
			return
		}
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	var err error
	if cacheEntry.CompressedContent != nil && strings.Contains(req.Header.Get("Accept-Encoding"), "gzip") {
		w.Header().Set("Content-Encoding", "gzip")
		_, err = w.Write(cacheEntry.CompressedContent)
	} else if cacheEntry.Content == nil && cacheEntry.CompressedContent != nil {
		//cacheEntry.Content might be empty when client does not accept gzip encoding
		//so we need to decompress Compressed Content
		gzipReader, _ := gzip.NewReader(bytes.NewReader(cacheEntry.CompressedContent))
		io.Copy(w, gzipReader)
		gzipReader.Close()
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

func (h *APIHandler) writeError(ctx context.Context, w http.ResponseWriter, err string, code int) {
	if h.callbacks.OnBeforeWriteResponse != nil {
		h.callbacks.OnBeforeWriteResponse(ctx, w)
	}

	if err == "" {
		err = http.StatusText(code)
	}
	http.Error(w, err, code)
}

func (h *APIHandler) writeInternalError(ctx context.Context, w http.ResponseWriter, err string) {
	if PrintDebug {
		h.writeError(ctx, w, http.StatusText(http.StatusInternalServerError)+":\n"+err, http.StatusInternalServerError)
	} else {
		h.writeError(ctx, w, "", http.StatusInternalServerError)
	}
}

func (h *APIHandler) IsDebug(req *http.Request) bool {
	return req.FormValue("debug") == "true"
}

func (h *APIHandler) writeTimeoutError(ctx context.Context, r *http.Request, w http.ResponseWriter) {
	err := &gorpc.CallHandlerError{
		Type: gorpc.ErrorReturnedFromCall,
		Err:  errors.New("Request timed out"),
	}
	if h.callbacks.OnError != nil {
		h.callbacks.OnError(ctx, w, r, nil, err)
	}
	h.writeError(ctx, w, err.UserMessage(), http.StatusServiceUnavailable)
}

func (h *APIHandler) SetTimeout(timeout time.Duration) *APIHandler {
	h.timeout = timeout
	return h
}
