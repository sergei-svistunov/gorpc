package http_json

import (
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

type httpSessionResponse struct {
	Result string      `json:"result"`
	Data   interface{} `json:"data"`
	Error  string      `json:"error"`
}

type APIHandlerCallbacks struct {
	/*
		OnInitCtx func(req *http.Request) context.Context
		OnSuccess func(ctx context.Context, handlerResponse interface{})
		OnError   func(ctx context.Context, err error)
		OnPanic   func(ctx context.Context, r interface{}, trace []byte)
	*/

	// OnInitCtx prepares context for handler (each time for handler call)
	OnInitCtx             func(req *http.Request) (context.Context, error)
	OnError               func(ctx context.Context, w http.ResponseWriter, req *http.Request, resp interface{}, err *gorpc.CallHandlerError)
	OnPanic               func(ctx context.Context, w http.ResponseWriter, r interface{}, trace []byte, req *http.Request)
	OnBeforeWriteResponse func(ctx context.Context, w http.ResponseWriter)
	OnSuccess             func(ctx context.Context, req *http.Request, handlerResponse interface{}, startTime time.Time)
	On404                 func(ctx context.Context, req *http.Request)
}

type APIHandler struct {
	hm        *gorpc.HandlersManager
	callbacks APIHandlerCallbacks
}

func NewAPIHandler(hm *gorpc.HandlersManager, callbacks APIHandlerCallbacks) *APIHandler {
	return &APIHandler{
		hm:        hm,
		callbacks: callbacks,
	}
}

func (h *APIHandler) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	startTime := time.Now()
	ctx := context.Background()

	defer func() {
		if r := recover(); r != nil {
			trace := make([]byte, 1024)
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
		var err error
		ctx, err = h.callbacks.OnInitCtx(req)
		if err != nil {
			er := &gorpc.CallHandlerError{
				Type: gorpc.ErrorReturnedFromCall,
				Err:  err,
			}
			if h.callbacks.OnError != nil {
				h.callbacks.OnError(ctx, w, req, nil, er)
			}
			resp.Result = "ERROR"
			resp.Data = er.UserMessage()
			resp.Error = er.ErrorCode()
			writeResponse(&resp, w, req)
			return
		}
	}

	handlerResponse, err := h.hm.CallHandler(ctx, handler, &ParametersGetter{Req: req})

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

	if h.callbacks.OnBeforeWriteResponse != nil {
		h.callbacks.OnBeforeWriteResponse(ctx, w)
	}

	if err := writeResponse(&resp, w, req); err != nil {
		if h.callbacks.OnError != nil {
			handlerError := &gorpc.CallHandlerError{
				Type: gorpc.ErrorWriteResponse,
				Err:  err,
			}
			h.callbacks.OnError(ctx, w, req, resp, handlerError)
		}
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	if h.callbacks.OnSuccess != nil {
		h.callbacks.OnSuccess(ctx, req, resp, startTime)
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
