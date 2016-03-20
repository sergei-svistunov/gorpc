package jsonrpc

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/sergei-svistunov/gorpc"
	"github.com/sergei-svistunov/gorpc/transport/cache"
	"github.com/stretchr/testify/suite"

	test_handler1 "github.com/sergei-svistunov/gorpc/test/handler1"
)

// Suite
type HttpJSONSute struct {
	suite.Suite

	server *httptest.Server
}

func (s *HttpJSONSute) SetupTest() {
	hm := gorpc.NewHandlersManager("github.com/sergei-svistunov/gorpc", gorpc.HandlersManagerCallbacks{})
	s.NoError(hm.RegisterHandler(test_handler1.NewHandler()))

	s.server = httptest.NewUnstartedServer(NewAPIHandler(hm, cache.NewMapCache(), APIHandlerCallbacks{}))
}

func TestRunHttpJSONSute(t *testing.T) {
	suite.Run(t, new(HttpJSONSute))
}

// Tests
func (s *HttpJSONSute) TestHttpJSON_CallWithRequiredArguments_Success() {
	s.server.Start()
	defer s.server.Close()

	body := &bytes.Buffer{}
	json.NewEncoder(body).Encode(map[string]interface{}{
		"jsonrpc": "2.0",
		"method":  "/test/handler1/v1",
		"params": map[string]interface{}{
			"req_int": 123,
		},
		"id": 1,
	})

	resp, err := http.Post(s.server.URL, "application/json", body)

	s.NoError(err)
	s.Equal(200, resp.StatusCode)

	respBody := &bytes.Buffer{}
	io.Copy(respBody, resp.Body)

	s.Contains(respBody.String(), `"result":{"string":"Test","int":123}`)
}
