package http_json

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/sergei-svistunov/gorpc"
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

	s.server = httptest.NewUnstartedServer(NewAPIHandler(hm, NewTestCache(), APIHandlerCallbacks{}))
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

// Benchmarks
func BenchmarkHttpJSON_CallWithRequiredArguments_Success(b *testing.B) {
	hm := gorpc.NewHandlersManager("github.com/sergei-svistunov/gorpc", gorpc.HandlersManagerCallbacks{})
	if err := hm.RegisterHandler(test_handler1.NewHandler()); err != nil {
		b.Fatal(err.Error())
	}

	handler := NewAPIHandler(hm, NewTestCache(), APIHandlerCallbacks{})
	request, _ := http.NewRequest("GET", "/test/handler1/v1/?req_int=123", nil)
	recorder := httptest.NewRecorder()

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		handler.ServeHTTP(recorder, request)
	}
}
