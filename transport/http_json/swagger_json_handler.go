package http_json

import (
	"net/http"

	"github.com/sergei-svistunov/gorpc"
)

type SwaggerJSONHandler struct {
	jsonB []byte
}

func NewSwaggerJSONHandler(hm *gorpc.HandlersManager, host string, apiPort uint16, callbacks SwaggerJSONCallbacks) *SwaggerJSONHandler {
	jsonB, err := GenerateSwaggerJSON(hm, host, apiPort, callbacks)
	if err != nil {
		panic(err)
	}
	return &SwaggerJSONHandler{jsonB}
}

func (h *SwaggerJSONHandler) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Write(h.jsonB)
}
