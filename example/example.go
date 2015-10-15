package main

import (
	"log"
	"net/http"

	"github.com/sergei-svistunov/gorpc"
	"github.com/sergei-svistunov/gorpc/swagger_ui"
	"github.com/sergei-svistunov/gorpc/transport/http_json"

	test_handler1 "github.com/sergei-svistunov/gorpc/test/handler1"

	"golang.org/x/net/context"
)

func main() {
	hm := gorpc.NewHandlersManager("github.com/sergei-svistunov/gorpc", gorpc.HandlersManagerCallbacks{})

	if err := hm.RegisterHandler(test_handler1.NewHandler()); err != nil {
		panic(err)
	}

	http.Handle("/", http_json.NewAPIHandler(hm, nil, http_json.APIHandlerCallbacks{
		OnError: func(ctx context.Context, w http.ResponseWriter, req *http.Request, resp interface{}, err *gorpc.CallHandlerError) {
			log.Println(err.Error())
		},
		OnPanic: func(ctx context.Context, w http.ResponseWriter, r interface{}, trace []byte, req *http.Request) {
			log.Println(r, "\n", string(trace))
		},
	}))
	http.Handle("/swagger.json", http_json.NewSwaggerJSONHandler(hm, http_json.SwaggerJSONCallbacks{}))
	http.Handle("/docs/", http.StripPrefix("/docs", swagger_ui.NewHTTPHandler()))

	http.ListenAndServe(":8080", nil)
}
