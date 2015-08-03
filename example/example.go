package main

import (
	"net/http"
	"time"

	"github.com/sergei-svistunov/gorpc"
	"github.com/sergei-svistunov/gorpc/swagger_ui"
	"github.com/sergei-svistunov/gorpc/transport/http_json"

	test_handler1 "github.com/sergei-svistunov/gorpc/test/handler1"
)

//  Cache implementation
type DummyCache struct{}

func (c *DummyCache) Get(key string) (interface{}, bool) {
	return nil, false
}

func (c *DummyCache) Put(key string, data interface{}, ttl time.Duration) {}

func main() {
	hm := gorpc.NewHandlersManager("github.com/sergei-svistunov/gorpc", &DummyCache{}, 0)

	if err := hm.RegisterHandler(test_handler1.NewHandler()); err != nil {
		panic(err)
	}

	http.Handle("/", http_json.NewAPIHandler(hm))
	http.Handle("/swagger.json", http_json.NewSwaggerJSONHandler(hm))
	http.Handle("/docs/", http.StripPrefix("/docs", swagger_ui.NewSwaggerUIHandler()))

	http.ListenAndServe(":8080", nil)
}
