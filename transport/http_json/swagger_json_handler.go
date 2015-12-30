package http_json

import (
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"strings"

	"github.com/sergei-svistunov/gorpc"
)

type SwaggerJSONHandler struct {
	apiPort   uint16
	hm        *gorpc.HandlersManager
	callbacks SwaggerJSONCallbacks
}

func NewSwaggerJSONHandler(hm *gorpc.HandlersManager, apiPort uint16, callbacks SwaggerJSONCallbacks) *SwaggerJSONHandler {
	return &SwaggerJSONHandler{apiPort, hm, callbacks}
}

func (h *SwaggerJSONHandler) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	var host string
	if h.apiPort != 0 {
		host = strings.Split(req.Header.Get("Host"), ":")[0] + strconv.FormatUint(uint64(h.apiPort), 10)
	}
	swagger, err := GenerateSwaggerJSON(h.hm, host, h.callbacks)
	if err != nil {
		log.Println(err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	json.NewEncoder(w).Encode(swagger)
}
