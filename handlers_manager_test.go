package gorpc

import (
	"errors"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"
	"golang.org/x/net/context"

	test_handler1 "github.com/sergei-svistunov/gorpc/test/handler1"
)

//  Cache implementation
type TestCache struct{}

func (c *TestCache) Get(key string) (interface{}, bool) {
	return nil, false
}

func (c *TestCache) Put(key string, data interface{}, ttl time.Duration) {}

// Parameters getter
type ParametersGetter struct {
	values map[string][]string
}

func (pg *ParametersGetter) IsExists(name string) bool {
	_, exists := pg.values[name]

	return exists
}

func (pg *ParametersGetter) GetStringSlice(name string) []string {
	name = strings.ToLower(name)
	if slice, ok := pg.values[name]; ok {
		return slice
	}

	return []string{}
}

func (pg *ParametersGetter) GetString(name string) (string, error) {
	return pg.get(name), nil
}

func (pg *ParametersGetter) GetBool(name string) (bool, error) {
	v, err := strconv.ParseBool(pg.get(name))
	if err != nil {
		err = errors.New(`Wrong value of param "` + name + `". It should be Bool`)
	}

	return v, err
}

func (pg *ParametersGetter) GetUint(name string) (uint, error) {
	v, err := strconv.ParseUint(pg.get(name), 0, 0)
	if err != nil {
		err = errors.New(`Wrong value of param "` + name + `". It should be Uint`)
	}

	return uint(v), err
}

func (pg *ParametersGetter) GetByte(name string) (uint8, error) {
	v, err := strconv.ParseUint(pg.get(name), 0, 8)
	if err != nil {
		err = errors.New(`Wrong value of param "` + name + `". It should be Uint`)
	}

	return uint8(v), err
}

func (pg *ParametersGetter) GetUint8(name string) (uint8, error) {
	v, err := strconv.ParseUint(pg.get(name), 0, 8)
	if err != nil {
		err = errors.New(`Wrong value of param "` + name + `". It should be Uint`)
	}

	return uint8(v), err
}

func (pg *ParametersGetter) GetUint16(name string) (uint16, error) {
	v, err := strconv.ParseUint(pg.get(name), 0, 16)
	if err != nil {
		err = errors.New(`Wrong value of param "` + name + `". It should be Uint`)
	}

	return uint16(v), err
}

func (pg *ParametersGetter) GetUint32(name string) (uint32, error) {
	v, err := strconv.ParseUint(pg.get(name), 0, 32)
	if err != nil {
		err = errors.New(`Wrong value of param "` + name + `". It should be Uint`)
	}

	return uint32(v), err
}

func (pg *ParametersGetter) GetUint64(name string) (uint64, error) {
	v, err := strconv.ParseUint(pg.get(name), 0, 64)
	if err != nil {
		err = errors.New(`Wrong value of param "` + name + `". It should be Uint`)
	}

	return v, err
}

func (pg *ParametersGetter) GetInt(name string) (int, error) {
	v, err := strconv.ParseInt(pg.get(name), 0, 0)
	if err != nil {
		err = errors.New(`Wrong value of param "` + name + `". It should be Int`)
	}

	return int(v), err
}

func (pg *ParametersGetter) GetInt8(name string) (int8, error) {
	v, err := strconv.ParseInt(pg.get(name), 0, 8)
	if err != nil {
		err = errors.New(`Wrong value of param "` + name + `". It should be Int`)
	}

	return int8(v), err
}

func (pg *ParametersGetter) GetInt16(name string) (int16, error) {
	v, err := strconv.ParseInt(pg.get(name), 0, 16)
	if err != nil {
		err = errors.New(`Wrong value of param "` + name + `". It should be Int`)
	}

	return int16(v), err
}

func (pg *ParametersGetter) GetInt32(name string) (int32, error) {
	v, err := strconv.ParseInt(pg.get(name), 0, 32)
	if err != nil {
		err = errors.New(`Wrong value of param "` + name + `". It should be Int`)
	}

	return int32(v), err
}

func (pg *ParametersGetter) GetInt64(name string) (int64, error) {
	v, err := strconv.ParseInt(pg.get(name), 0, 64)
	if err != nil {
		err = errors.New(`Wrong value of param "` + name + `". It should be Int`)
	}

	return v, err
}

func (pg *ParametersGetter) GetFloat32(name string) (float32, error) {
	v, err := strconv.ParseFloat(pg.get(name), 32)
	if err != nil {
		err = errors.New(`Wrong value of param "` + name + `". It should be Float`)
	}

	return float32(v), err
}

func (pg *ParametersGetter) GetFloat64(name string) (float64, error) {
	v, err := strconv.ParseFloat(pg.get(name), 64)
	if err != nil {
		err = errors.New(`Wrong value of param "` + name + `". It should be Float`)
	}

	return v, err
}

func (pg *ParametersGetter) get(name string) string {
	slice := pg.values[name]

	return slice[0]
}

// Suite definition
type HandlersManagerSuite struct {
	suite.Suite
	hm *HandlersManager
}

func (s *HandlersManagerSuite) SetupTest() {
	s.hm = NewHandlersManager("github.com/sergei-svistunov/gorpc", HandlersManagerCallbacks{}, &TestCache{}, 0)

	s.NoError(s.hm.RegisterHandler(test_handler1.NewHandler()))
}

func TestRunSuite(t *testing.T) {
	suite.Run(t, new(HandlersManagerSuite))
}

// Tests
func (s *HandlersManagerSuite) TestHandlerManager_CheckHandlersPaths() {
	s.Equal([]string{"/test/handler1"}, s.hm.GetHandlersPaths())
}

func (s *HandlersManagerSuite) TestHandlerManager_FindExistsHandler() {
	s.NotNil(s.hm.FindHandler("/test/handler1", 1))
}

func (s *HandlersManagerSuite) TestHandlerManager_CheckHandler1Struct() {
	hv := s.hm.FindHandler("/test/handler1", 1)

	s.Equal("v1", hv.Version)
	s.True(hv.UseCache)
	s.Equal([]handlerParameter{
		handlerParameter{
			Name:        "ReqInt",
			Type:        "int",
			Description: "Required integer argument",
			Key:         "req_int",
			IsRequired:  true,
			RawType:     hv.Parameters[0].RawType,
			getMethod:   hv.Parameters[0].getMethod,
			structField: hv.Parameters[0].structField,
		},
		handlerParameter{
			Name:        "Int",
			Type:        "int",
			Description: "Unrequired integer argument",
			Key:         "int",
			IsRequired:  false,
			RawType:     hv.Parameters[1].RawType,
			getMethod:   hv.Parameters[1].getMethod,
			structField: hv.Parameters[1].structField,
		},
	}, hv.Parameters)
}

func (s *HandlersManagerSuite) TestHandlerManager_CallHandler1V1_ReturnResult() {
	pg := &ParametersGetter{
		map[string][]string{
			"req_int": []string{"123"},
		},
	}
	res, err := s.hm.CallHandler(context.TODO(), s.hm.FindHandler("/test/handler1", 1), pg)

	s.NoError(err)
	s.Equal(&test_handler1.V1Res{"Test", 123}, res)
}

func (s *HandlersManagerSuite) TestHandlerManager_CallHandler1V1WuthoutReqArg_ReturnError() {
	pg := &ParametersGetter{
		map[string][]string{},
	}
	_, err := s.hm.CallHandler(context.TODO(), s.hm.FindHandler("/test/handler1", 1), pg)

	s.Error(err)
}
