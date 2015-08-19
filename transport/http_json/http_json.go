package http_json

import (
	"compress/gzip"
	"encoding/json"
	"net/http"
	"runtime"
	"strings"

	"github.com/sergei-svistunov/gorpc"
	"golang.org/x/net/context"
)

type httpSessionResponse struct {
	Result string      `json:"result"`
	Data   interface{} `json:"data"`
	Error  string      `json:"error"`
}

type APIHandlerCallbacks struct {
	OnInitCtx func(path string) context.Context
	OnOk      func(ctx context.Context, handlerResponse interface{})
	OnError   func(ctx context.Context, err error)
	OnPanic   func(ctx context.Context, r interface{}, trace []byte)
}

type APIHandler struct {
	hm        *gorpc.HandlersManager
	callbacks APIHandlerCallbacks
}

func NewAPIHandler(hm *gorpc.HandlersManager, callbacks APIHandlerCallbacks) *APIHandler {
	return &APIHandler{
		hm: hm,
	}
}

func (h *APIHandler) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	var ctx context.Context

	defer func() {
		if r := recover(); r != nil {
			trace := make([]byte, 1024)
			n := runtime.Stack(trace, false)
			trace = trace[:n]

			if h.callbacks.OnPanic != nil {
				h.callbacks.OnPanic(ctx, r, trace)
			}

			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		}
	}()

	if req.Method != "POST" && req.Method != "GET" {
		http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)

		return
	}

	req.ParseForm()

	path := req.URL.Path
	handler := h.hm.FindHandlerByRoute(path)

	if handler == nil {
		http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
		return
	}

	if h.callbacks.OnInitCtx != nil {
		ctx = h.callbacks.OnInitCtx(path)
	} else {
		ctx = context.TODO()
	}

	handlerResponse, err := h.hm.CallHandler(ctx, handler, &ParametersGetter{Req: req})

	var resp httpSessionResponse
	if err == nil {
		if h.callbacks.OnOk != nil {
			h.callbacks.OnOk(ctx, handlerResponse)
		}

		resp.Result = "OK"
		resp.Data = handlerResponse
	} else {
		if h.callbacks.OnError != nil {
			h.callbacks.OnError(ctx, err)
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

	if err := writeResponse(&resp, w, req); err != nil {
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)

		return
	}
}

func writeResponse(resp *httpSessionResponse, w http.ResponseWriter, req *http.Request) error {
	payload, err := json.Marshal(resp)
	if err != nil {
		return err
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	if len(payload) > 1024 && strings.Contains(req.Header.Get("Accept-Encoding"), "gzip") {
		w.Header().Set("Content-Encoding", "gzip")
		gzipWriter := gzip.NewWriter(w)
		if _, err := gzipWriter.Write(payload); err != nil {
			return err
		}
		return gzipWriter.Close()
	}

	_, err = w.Write(payload)
	return err
}

func (h *APIHandler) CanServe(req *http.Request) bool {
    path := req.URL.Path
    handler := h.hm.FindHandlerByRoute(path)
    if handler == nil {
        return false
    }
    return true
}