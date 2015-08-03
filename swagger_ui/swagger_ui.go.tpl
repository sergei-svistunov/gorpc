package swagger_ui

import (
	"encoding/base64"
	"mime"
	"net/http"
	"strings"
)

type SwaggerUIHandler struct {
	files map[string]string
}

func NewSwaggerUIHandler() *SwaggerUIHandler {
	return &SwaggerUIHandler{
		files: map[string]string{
		/* FILES */
		},
	}
}

func (h *SwaggerUIHandler) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	fileName := req.URL.Path
	if fileName == "/" {
		fileName = "/index.html"
	}

	file, exists := h.files[fileName]
	if !exists || strings.LastIndex(fileName, ".") < 1 {
		http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
	}

	w.Header().Set("Content-Type", mime.TypeByExtension(fileName[strings.LastIndex(fileName, "."):]))

	content, _ := base64.StdEncoding.DecodeString(file)
	w.Write(content)
}
