package http_json

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"errors"
	"net/http"
	"runtime"
	"strings"
	"time"

	"github.com/sergei-svistunov/gorpc"
	"golang.org/x/net/context"
)

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

func newContext(parent context.Context, info *requestInfo) context.Context {
	return context.WithValue(parent, requestInfoKey, info)
}

func EnableCacheInTransport(ctx context.Context) {
	if info, ok := fromContext(ctx); ok {
		info.UseCache = true
	}
}

func EnableETag(ctx context.Context) {
	if info, ok := fromContext(ctx); ok {
		info.UseCache = true
		info.UseETag = true
	}
}

type httpSessionResponse struct {
	Result string      `json:"result"`
	Data   interface{} `json:"data"`
	Error  string      `json:"error"`
}

type APIHandlerCallbacks struct {
	OnInitCtx             func(req *http.Request) context.Context
	OnError               func(ctx context.Context, w http.ResponseWriter, req *http.Request, resp interface{}, err *gorpc.CallHandlerError)
	OnPanic               func(ctx context.Context, w http.ResponseWriter, r interface{}, trace []byte, req *http.Request)
	OnStartServing        func(req *http.Request)
	OnEndServing          func(req *http.Request, startTime time.Time)
	OnBeforeWriteResponse func(ctx context.Context, w http.ResponseWriter)
	OnSuccess             func(ctx context.Context, req *http.Request, handlerResponse interface{}, startTime time.Time)
	On404                 func(ctx context.Context, req *http.Request)
	OnCacheHit            func(ctx context.Context, entry *CacheEntry)
	OnCacheMiss           func(ctx context.Context)
	GetCacheKey           func(ctx context.Context, req *http.Request, params interface{}) []byte
}

type APIHandler struct {
	hm        *gorpc.HandlersManager
	cache     ICache
	callbacks APIHandlerCallbacks
}

func NewAPIHandler(hm *gorpc.HandlersManager, cache ICache, callbacks APIHandlerCallbacks) *APIHandler {
	return &APIHandler{
		hm:        hm,
		cache:     cache,
		callbacks: callbacks,
	}
}

func (h *APIHandler) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	startTime := time.Now()
	ctx := context.Background()

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

	req.ParseForm()

	path := req.URL.Path
	handler := h.hm.FindHandlerByRoute(path)

	if handler == nil {
		if h.callbacks.On404 != nil {
			h.callbacks.On404(ctx, req)
		}
		h.writeError(ctx, w, "", http.StatusNotFound)
		return
	}

	var resp httpSessionResponse
	if h.callbacks.OnInitCtx != nil {
		ctx = h.callbacks.OnInitCtx(req)
	}
	requestInfo := &requestInfo{}
	ctx = newContext(ctx, requestInfo)

	jsonRequest := strings.HasPrefix(req.Header.Get("Content-Type"), "application/json")
	if jsonRequest && req.Method != "POST" {
		if h.callbacks.OnError != nil {
			err := &gorpc.CallHandlerError{
				Type: gorpc.ErrorInvalidMethod,
				Err:  errors.New(http.StatusText(http.StatusBadRequest)),
			}
			h.callbacks.OnError(ctx, w, req, nil, err)
		}
		h.writeError(ctx, w, "", http.StatusBadRequest)
		return
	}

	var paramsGetter gorpc.IHandlerParameters
	if jsonRequest {
		paramsGetter = &JsonParametersGetter{Req: req.Body}
	} else {
		paramsGetter = &ParametersGetter{Req: req}
	}
	params, paramsErr := h.hm.UnmarshalParameters(ctx, handler, paramsGetter)
	if paramsErr != nil {
		if h.callbacks.OnError != nil {
			grpcErr := &gorpc.CallHandlerError{
				Type: gorpc.ErrorInParameters,
				Err:  paramsErr,
			}
			h.callbacks.OnError(ctx, w, req, resp, grpcErr)
		}
		h.writeError(ctx, w, paramsErr.Error(), http.StatusBadRequest)
		return
	}

	var cacheKey []byte
	var cacheEntry *CacheEntry
	if h.cache != nil {
		if h.callbacks.GetCacheKey != nil {
			cacheKey = h.callbacks.GetCacheKey(ctx, req, params.Interface())
		} else {
			var err error
			cacheKey, err = json.Marshal(params.Interface())
			if err != nil {
				// TODO: call callback.onError?
				cacheKey = nil
			}
		}
		if cacheKey != nil {
			cacheEntry = h.cache.Get(cacheKey)
			if cacheEntry != nil {
				if h.callbacks.OnCacheHit != nil {
					h.callbacks.OnCacheHit(ctx, cacheEntry)
				}
				h.writeResponse(ctx, startTime, cacheEntry, resp, w, req)
				return
			}

			if h.callbacks.OnCacheMiss != nil {
				h.callbacks.OnCacheMiss(ctx)
			}
		}
	}

	handlerResponse, err := h.hm.CallHandler(ctx, handler, params)

	if err == nil {
		resp.Result = "OK"
		resp.Data = handlerResponse
	} else {
		if h.callbacks.OnError != nil {
			h.callbacks.OnError(ctx, w, req, resp, err)
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

	cacheEntry = &CacheEntry{}
	var jerr error
	cacheEntry.Content, jerr = json.Marshal(resp)
	if jerr != nil {
		if h.callbacks.OnError != nil {
			handlerError := &gorpc.CallHandlerError{
				Type: gorpc.ErrorWriteResponse,
				Err:  jerr,
			}
			h.callbacks.OnError(ctx, w, req, resp, handlerError)
		}
		h.writeError(ctx, w, "", http.StatusInternalServerError)
		return
	}
	if len(cacheEntry.Content) > 1024 && (cacheKey != nil || strings.Contains(req.Header.Get("Accept-Encoding"), "gzip")) {
		buf := bytes.NewBuffer(cacheEntry.CompressedContent)
		gzip.NewWriter(buf).Write(cacheEntry.Content)
	}

	if requestInfo.UseCache && h.cache != nil && cacheKey != nil {
		if requestInfo.UseETag {
			cacheEntry.hash, _ = etagHash(cacheEntry.Content)
		}
		h.cache.Put(cacheKey, cacheEntry)
	}

	h.writeResponse(ctx, startTime, cacheEntry, resp, w, req)
}

func (h *APIHandler) CanServe(req *http.Request) bool {
	path := req.URL.Path
	handler := h.hm.FindHandlerByRoute(path)
	return handler != nil
}

func (h *APIHandler) writeResponse(ctx context.Context, startTime time.Time, cacheEntry *CacheEntry, resp httpSessionResponse,
	w http.ResponseWriter, req *http.Request) {

	if h.callbacks.OnSuccess != nil {
		h.callbacks.OnSuccess(ctx, req, resp, startTime)
	}

	if h.callbacks.OnBeforeWriteResponse != nil {
		h.callbacks.OnBeforeWriteResponse(ctx, w)
	}

	if cacheEntry.hash != "" {
		w.Header().Set("Etag", cacheEntry.hash)
		if cacheEntry.hash == req.Header.Get("If-None-Match") {
			if h.callbacks.OnSuccess != nil {
				h.callbacks.OnSuccess(ctx, req, resp, startTime)
			}
			if h.callbacks.OnBeforeWriteResponse != nil {
				h.callbacks.OnBeforeWriteResponse(ctx, w)
			}
			w.WriteHeader(http.StatusNotModified)
			return
		}
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
