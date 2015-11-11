package adapter

import (
	"github.com/sergei-svistunov/gorpc"
	"net/http"
)

type AdapterHandler struct {
	hm *gorpc.HandlersManager
}

func NewHandler(hm *gorpc.HandlersManager) *AdapterHandler {
	return &AdapterHandler{hm}
}

func (h *AdapterHandler) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	var (
		pkgName     string
		serviceName string
	)

	if err := req.ParseForm(); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if pkg := req.Form.Get("package"); pkg != "" {
		pkgName = pkg
	}
	if srvName := req.Form.Get("service_name"); srvName != "" {
		serviceName = srvName
	}

	generator := NewHttpJsonLibGenerator(h.hm, pkgName, serviceName)

	code, err := generator.Generate()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.Write(code)
}
