package http_json

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/sergei-svistunov/gorpc"
	"github.com/stretchr/testify/suite"

	test_handler1 "github.com/sergei-svistunov/gorpc/test/handler1"
)

//  Cache implementation
type TestCache struct{}

func (c *TestCache) Get(key string) (interface{}, bool) {
	return nil, false
}

func (c *TestCache) Put(key string, data interface{}, ttl time.Duration) {}

// Suite
type HttpJSONSute struct {
	suite.Suite

	server *httptest.Server
}

func (s *HttpJSONSute) SetupTest() {
	hm := gorpc.NewHandlersManager("github.com/sergei-svistunov/gorpc", &TestCache{}, 0)
	s.NoError(hm.RegisterHandler(test_handler1.NewHandler()))

	s.server = httptest.NewUnstartedServer(NewAPIHandler(hm))
}

func TestRunHttpJSONSute(t *testing.T) {
	suite.Run(t, new(HttpJSONSute))
}

// Tests
func (s *HttpJSONSute) TestHttpJSON_CallWithRequiredArguments_Success() {
	s.server.Start()
	defer s.server.Close()

	resp, err := http.Get(s.server.URL + "/test/handler1/v1/?req_int=123")

	s.NoError(err)
	s.Equal(200, resp.StatusCode)
}

func (s *HttpJSONSute) TestHttpJSON_CallWithoutRequiredArguments_BadRequest() {
	s.server.Start()
	defer s.server.Close()

	resp, err := http.Get(s.server.URL + "/test/handler1/v1/")

	s.NoError(err)
	s.Equal(400, resp.StatusCode)
}