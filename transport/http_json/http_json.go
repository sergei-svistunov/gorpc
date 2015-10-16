package http_json

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"runtime"
	"strings"
	"time"

	"github.com/sergei-svistunov/gorpc"
	"golang.org/x/net/context"
)

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

			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
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
		http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
		return
	}

	req.ParseForm()

	path := req.URL.Path
	handler := h.hm.FindHandlerByRoute(path)

	if handler == nil {
		if h.callbacks.On404 != nil {
			h.callbacks.On404(ctx, req)
		}
		http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
		return
	}

	var resp httpSessionResponse
	if h.callbacks.OnInitCtx != nil {
		ctx = h.callbacks.OnInitCtx(req)
	}

	jsonRequest := (req.Header.Get("Content-Type") == "application/json")
	if jsonRequest && req.Method != "POST" {
		if h.callbacks.OnError != nil {
			err := &gorpc.CallHandlerError{
				Type: gorpc.ErrorInvalidMethod,
				Err:  errors.New(http.StatusText(http.StatusBadRequest)),
			}
			h.callbacks.OnError(ctx, w, req, nil, err)
		}
		http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return
	}

	var paramsGetter gorpc.IHandlerParameters
	if jsonRequest {
		paramsGetter = &JsonParametersGetter{Req: req}
	} else {
		paramsGetter = &ParametersGetter{Req: req}
	}
	params, paramsErr := h.hm.PrepareParameters(ctx, handler, paramsGetter)
	if paramsErr != nil {
		if h.callbacks.OnError != nil {
			grpcErr := &gorpc.CallHandlerError{
				Type: gorpc.ErrorInParameters,
				Err:  paramsErr,
			}
			h.callbacks.OnError(ctx, w, req, resp, grpcErr)
		}
		http.Error(w, paramsErr.Error(), http.StatusBadRequest)
		return
	}

	var cacheKey []byte
	var cacheEntry *CacheEntry
	if h.cache != nil && handler.UseCache {
		if h.callbacks.GetCacheKey != nil {
			cacheKey = h.callbacks.GetCacheKey(ctx, req, params.Interface())
		} else {
			var err error
			cacheKey, err = json.Marshal(params.Interface())
			if err != nil {
				log.Print(err.Error())
				cacheKey = nil
			}
		}
		if cacheKey != nil {
			cacheEntry = h.cache.Get(cacheKey)
			if cacheEntry != nil {
				if h.callbacks.OnCacheHit != nil {
					h.callbacks.OnCacheHit(ctx, cacheEntry)
				}
			} else {
				if h.callbacks.OnCacheMiss != nil {
					h.callbacks.OnCacheMiss(ctx)
				}
			}
		}
	}

	if cacheEntry != nil {
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
		}

		return
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
			http.Error(w, err.UserMessage(), http.StatusBadRequest)
			return
		default:
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
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
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}
	if len(cacheEntry.Content) > 1024 && (cacheKey != nil || strings.Contains(req.Header.Get("Accept-Encoding"), "gzip")) {
		buf := bytes.NewBuffer(cacheEntry.CompressedContent)
		gzip.NewWriter(buf).Write(cacheEntry.Content)
	}

	if h.cache != nil && cacheKey != nil {
		h.cache.Put(cacheKey, cacheEntry)
	}

	if h.callbacks.OnSuccess != nil {
		h.callbacks.OnSuccess(ctx, req, resp, startTime)
	}

	if h.callbacks.OnBeforeWriteResponse != nil {
		h.callbacks.OnBeforeWriteResponse(ctx, w)
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	var werr error
	if cacheEntry.CompressedContent != nil && strings.Contains(req.Header.Get("Accept-Encoding"), "gzip") {
		w.Header().Set("Content-Encoding", "gzip")
		_, werr = w.Write(cacheEntry.CompressedContent)
	} else {
		_, werr = w.Write(cacheEntry.Content)
	}
	if werr != nil && h.callbacks.OnError != nil {
		handlerError := &gorpc.CallHandlerError{
			Type: gorpc.ErrorWriteResponse,
			Err:  werr,
		}
		h.callbacks.OnError(ctx, w, req, resp, handlerError)
	}
}

func (h *APIHandler) CanServe(req *http.Request) bool {
	path := req.URL.Path
	handler := h.hm.FindHandlerByRoute(path)
	if handler == nil {
		return false
	}
	return true
}
