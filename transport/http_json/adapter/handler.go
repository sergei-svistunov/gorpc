package adapter

import (
	"github.com/sergei-svistunov/gorpc"
	"net/http"
)

type AdapterHandler struct {
	hm   *gorpc.HandlersManager
	code []byte
}

func NewJSONClientLibGeneratorHandler(hm *gorpc.HandlersManager) *AdapterHandler {
	return &AdapterHandler{
		hm: hm,
	}
}

func (h *AdapterHandler) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	var (
		pkgName     string
		serviceName string
	)

	if err := req.ParseForm(); err != nil {
		w.Write([]byte(err.Error()))
	}
	if pkg := req.Form.Get("package"); pkg != "" {
		pkgName = pkg
	}
	if srvName := req.Form.Get("service_name"); srvName != "" {
		serviceName = srvName
	}

	generator := NewHttpJsonLibGenerator(h.hm, pkgName, serviceName)

	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	code, err := generator.Generate()
	if err != nil {
		w.Write([]byte(err.Error()))
		return
	}
	w.Write(code)
}
