package swagger_ui

import (
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/suite"
)

// Suite
type SwaggerUISute struct {
	suite.Suite

	server *httptest.Server
}

func (s *SwaggerUISute) SetupTest() {
	s.server = httptest.NewUnstartedServer(NewHTTPHandler())
}

func TestRunSwaggerUISute(t *testing.T) {
	suite.Run(t, new(SwaggerUISute))
}

// Tests
func (s *SwaggerUISute) TestSwaggerUI_GetIndex_Success() {
	s.server.Start()
	defer s.server.Close()

	resp, err := http.Get(s.server.URL + "/index.html")

	s.NoError(err)
	s.Equal(200, resp.StatusCode)

	body, err := ioutil.ReadAll(resp.Body)
	s.NoError(err)
	s.NotEmpty(body)
}
