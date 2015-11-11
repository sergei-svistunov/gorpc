package main

import (
	"log"
	"net/http"

	"github.com/sergei-svistunov/gorpc"
	"github.com/sergei-svistunov/gorpc/swagger_ui"
	"github.com/sergei-svistunov/gorpc/transport/http_json"
	http_json_adapter "github.com/sergei-svistunov/gorpc/transport/http_json/adapter"
	"github.com/sergei-svistunov/gorpc/example/client"
	test_handler1 "github.com/sergei-svistunov/gorpc/test/handler1"

	"golang.org/x/net/context"
)

//go:generate curl "http://localhost:8080/client.go?service_name=example&package=client" --output client/client.go --create-dirs --silent --show-error

func main() {
	hm := gorpc.NewHandlersManager("github.com/sergei-svistunov/gorpc", gorpc.HandlersManagerCallbacks{})

	hm.MustRegisterHandler(test_handler1.NewHandler())

	// API
	http.Handle("/", http_json.NewAPIHandler(hm, nil, http_json.APIHandlerCallbacks{
		OnError: func(ctx context.Context, w http.ResponseWriter, req *http.Request, resp interface{}, err *gorpc.CallHandlerError) {
			log.Println(err.Error())
		},
		OnPanic: func(ctx context.Context, w http.ResponseWriter, r interface{}, trace []byte, req *http.Request) {
			log.Println(r, "\n", string(trace))
		},
	}))

	// Docs
	http.Handle("/swagger.json", http_json.NewSwaggerJSONHandler(hm, http_json.SwaggerJSONCallbacks{}))
	http.Handle("/docs/", http.StripPrefix("/docs", swagger_ui.NewHTTPHandler()))

	// Client SDK
	http.Handle("/client.go", http_json_adapter.NewHandler(hm))

	err := http.ListenAndServe(":8080", nil)
	if err != nil {
		log.Fatal(err)
	}

	e := client.NewExample(nil, 0)
	e.TestHandler1V1(context.Background(), client.TestHandler1V1Args{})
}
