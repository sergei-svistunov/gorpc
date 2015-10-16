package main

import (
	"fmt"
	"net/http"

	"github.com/sergei-svistunov/gorpc"
	"github.com/sergei-svistunov/gorpc/swagger_ui"
	"github.com/sergei-svistunov/gorpc/transport/http_json"

	test_handler1 "github.com/sergei-svistunov/gorpc/test/handler1"
)

func main() {
	hm := gorpc.NewHandlersManager("github.com/sergei-svistunov/gorpc", gorpc.HandlersManagerCallbacks{})

	if err := hm.RegisterHandler(test_handler1.NewHandler()); err != nil {
		panic(err)
	}

	http.Handle("/", http_json.NewAPIHandler(hm, nil, http_json.APIHandlerCallbacks{}))
	http.Handle("/swagger.json", http_json.NewSwaggerJSONHandler(hm, http_json.SwaggerJSONCallbacks{}))
	http.Handle("/docs/", http.StripPrefix("/docs", swagger_ui.NewHTTPHandler()))

	err := http.ListenAndServe(":8080", nil)
	if err != nil {
		fmt.Println(err)
	}
}
