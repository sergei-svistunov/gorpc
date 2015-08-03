package http_json

import (
	"compress/gzip"
	"encoding/json"
	"net/http"
	"regexp"
	"runtime"
	"strconv"
	"strings"

	"github.com/sergei-svistunov/gorpc"
	"golang.org/x/net/context"
)

type httpSessionResponse struct {
	Result string      `json:"result"`
	Data   interface{} `json:"data"`
	Error  string      `json:"error"`
}

type APIHandler struct {
	hm     *gorpc.HandlersManager
	pathRe *regexp.Regexp
}

func NewAPIHandler(hm *gorpc.HandlersManager) *APIHandler {
	pathRe := regexp.MustCompile(`^(.+?)/v(\d+)/?$`)

	return &APIHandler{
		hm:     hm,
		pathRe: pathRe,
	}
}

func (h *APIHandler) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	if req.Method != "POST" && req.Method != "GET" {
		http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)

		return
	}

	req.ParseForm()

	path := req.URL.Path

	defer func() {
		if r := recover(); r != nil {
			trace := make([]byte, 1024)
			n := runtime.Stack(trace, false)
			trace = trace[:n]

			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		}
	}()

	matches := h.pathRe.FindStringSubmatch(path)
	if len(matches) != 3 {
		return
	}

	handlerPath := matches[1]
	version, _ := strconv.Atoi(matches[2])
	handler := h.hm.FindHandler(handlerPath, version)

	if handler == nil {
		http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
		return
	}

	ctx := context.TODO()
	handlerResponse, err := h.hm.CallHandler(ctx, handler, &ParametersGetter{Req: req})

	var resp httpSessionResponse
	if err == nil {
		resp.Result = "OK"
		resp.Data = handlerResponse
	} else {
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
