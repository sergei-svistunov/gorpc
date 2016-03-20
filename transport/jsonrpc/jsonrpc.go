package jsonrpc

import (
	"encoding/json"
	"fmt"
	"hash/crc32"
	"net/http"
	"runtime"
	"time"

	"github.com/sergei-svistunov/gorpc"
	"github.com/sergei-svistunov/gorpc/transport/cache"
	"golang.org/x/net/context"
)

var PrintDebug = false

type APIHandlerCallbacks struct {
	OnInitCtx func(ctx context.Context, req *http.Request) context.Context
	//OnError               func(ctx context.Context, w http.ResponseWriter, req *http.Request, resp interface{}, err *gorpc.CallHandlerError)
	OnPanic               func(ctx context.Context, w http.ResponseWriter, r interface{}, trace []byte, req *http.Request)
	OnStartServing        func(ctx context.Context, req *http.Request)
	OnEndServing          func(ctx context.Context, req *http.Request, startTime time.Time)
	OnBeforeWriteResponse func(ctx context.Context, w http.ResponseWriter)
	//OnSuccess             func(ctx context.Context, req *http.Request, handlerResponse interface{}, startTime time.Time)
	//On404                 func(ctx context.Context, req *http.Request)
	//OnCacheHit            func(ctx context.Context, entry *cache.CacheEntry)
	//OnCacheMiss           func(ctx context.Context)
	//GetCacheKey           func(ctx context.Context, req *http.Request, params interface{}) []byte
}

type APIHandler struct {
	hm        *gorpc.HandlersManager
	cache     cache.ICache
	callbacks APIHandlerCallbacks
}

type jsonRpcResponse struct {
	JSONRPC string              `json:"jsonrpc"`
	Result  interface{}         `json:"result,omitempty"`
	Error   *jsonRpcResponseErr `json:"error,omitempty"`
	Id      interface{}         `json:"id,omitempty"`
	Debug   interface{}         `json:"debug,omitempty"`
}

type jsonRpcResponseErr struct {
	Code    int64       `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
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

	w.Header().Set("Content-Type", "application/json-rpc")

	if req.Body != nil {
		defer req.Body.Close()
	}

	ctx := context.Background()
	if h.callbacks.OnInitCtx != nil {
		ctx = h.callbacks.OnInitCtx(ctx, req)
	}
	ctx = cache.NewContext(ctx)

	if h.callbacks.OnStartServing != nil {
		h.callbacks.OnStartServing(ctx, req)
	}

	defer func() {
		if h.callbacks.OnEndServing != nil {
			h.callbacks.OnEndServing(ctx, req, startTime)
		}

		if r := recover(); r != nil {
			trace := make([]byte, 16*1024)
			n := runtime.Stack(trace, false)
			trace = trace[:n]

			if h.callbacks.OnPanic != nil {
				h.callbacks.OnPanic(ctx, w, r, trace, req)
			}
			var debugInfo *string
			if PrintDebug {
				info := fmt.Sprintf("%#v", r) + "\n\n" + string(trace)
				debugInfo = &info
			}
			h.writeError(ctx, w, -32603, "Internal error", debugInfo)
		}
	}()

	if req.Method != "POST" {
		h.writeHttpError(ctx, w, http.StatusMethodNotAllowed)
		return
	}

	var rpcReq struct {
		JSONRPC string      `json:"jsonrpc"`
		Method  string      `json:"method"`
		Params  interface{} `json:"params"`
		Id      interface{} `json:"id"`
	}

	// ToDo: Check Content-Length
	decoder := json.NewDecoder(req.Body)
	decoder.UseNumber()

	if err := decoder.Decode(&rpcReq); err != nil {
		h.writeError(ctx, w, -32700, "Parse error", err.Error())
		return
	}

	if rpcReq.JSONRPC != "2.0" {
		h.writeError(ctx, w, -32600, "Invalid Request", "Field 'jsonrpc' must be '2.0'")
		return
	}

	if rpcReq.Id == nil {
		h.writeError(ctx, w, -32600, "Invalid Request", "Not field 'id'")
		return
	}

	handler := h.hm.FindHandlerByRoute(rpcReq.Method)
	if handler == nil {
		h.writeError(ctx, w, -32601, "Method not found", nil)
		return
	}

	params, err := h.hm.UnmarshalParameters(ctx, handler, &ParametersGetter{values: rpcReq.Params})
	if err != nil {
		h.writeError(ctx, w, -32602, "Invalid params", err.Error())
		return
	}

	handlerResponse, hmErr := h.hm.CallHandler(ctx, handler, params)
	if err != nil {
		if hmErr.Type == gorpc.ErrorReturnedFromCall {
			h.writeError(ctx, w, -32000-int64(crc32.ChecksumIEEE([]byte(hmErr.ErrorCode()))%100), hmErr.ErrorCode(), hmErr.UserMessage())
		} else {
			if PrintDebug {
				h.writeError(ctx, w, -32603, "Internal error", hmErr.Error())
			} else {
				h.writeError(ctx, w, -32603, "Internal error", nil)
			}
		}
		return
	}

	h.writeResponse(ctx, w, handlerResponse)
}

func (h *APIHandler) writeResponse(ctx context.Context, w http.ResponseWriter, data interface{}) {
	if h.callbacks.OnBeforeWriteResponse != nil {
		h.callbacks.OnBeforeWriteResponse(ctx, w)
	}

	resp := jsonRpcResponse{
		JSONRPC: "2.0",
		Result:  data,
	}

	json.NewEncoder(w).Encode(resp)
}

func (h *APIHandler) writeError(ctx context.Context, w http.ResponseWriter, code int64, message string, data interface{}) {
	if h.callbacks.OnBeforeWriteResponse != nil {
		h.callbacks.OnBeforeWriteResponse(ctx, w)
	}

	respErr := &jsonRpcResponseErr{
		Code:    code,
		Message: message,
		Data:    data,
	}

	resp := jsonRpcResponse{
		JSONRPC: "2.0",
		Error:   respErr,
	}

	json.NewEncoder(w).Encode(resp)
}

func (h *APIHandler) writeHttpError(ctx context.Context, w http.ResponseWriter, code int) {
	if h.callbacks.OnBeforeWriteResponse != nil {
		h.callbacks.OnBeforeWriteResponse(ctx, w)
	}

	http.Error(w, http.StatusText(code), code)
}
