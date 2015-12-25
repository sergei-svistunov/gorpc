package http_json

import (
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/sergei-svistunov/gorpc"
	"github.com/stretchr/testify/suite"

	test_handler1 "github.com/sergei-svistunov/gorpc/test/handler1"
)

// Suite
type SwaggerJSONSute struct {
	suite.Suite

	server *httptest.Server
}

func (s *SwaggerJSONSute) SetupTest() {
	hm := gorpc.NewHandlersManager("github.com/sergei-svistunov/gorpc", gorpc.HandlersManagerCallbacks{})
	s.NoError(hm.RegisterHandler(test_handler1.NewHandler()))

	s.server = httptest.NewUnstartedServer(NewSwaggerJSONHandler(hm, "", 0, SwaggerJSONCallbacks{}))
}

func TestRunSwaggerJSONSute(t *testing.T) {
	suite.Run(t, new(SwaggerJSONSute))
}

// Tests
func (s *SwaggerJSONSute) TestSwaggerJSON_Call_Success() {
	s.server.Start()
	defer s.server.Close()

	resp, err := http.Get(s.server.URL)

	s.NoError(err)
	s.Equal(200, resp.StatusCode)

	body, err := ioutil.ReadAll(resp.Body)
	s.NoError(err)
	s.NotEmpty(body)
}
