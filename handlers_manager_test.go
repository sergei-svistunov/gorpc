package gorpc

import (
	"encoding/json"
	"fmt"
	"reflect"
	"testing"

	"github.com/stretchr/testify/suite"
	"golang.org/x/net/context"

	test_handler1 "github.com/sergei-svistunov/gorpc/test/handler1"
	test_handler_foreign_arguments "github.com/sergei-svistunov/gorpc/test/handler_foreign_arguments"
	test_handler_foreign_return_values "github.com/sergei-svistunov/gorpc/test/handler_foreign_return_values"
)

// Suite definition
type HandlersManagerSuite struct {
	suite.Suite
	hm *HandlersManager
}

func (s *HandlersManagerSuite) SetupTest() {
	s.hm = NewHandlersManager("github.com/sergei-svistunov/gorpc", HandlersManagerCallbacks{})

	s.NoError(s.hm.RegisterHandler(test_handler1.NewHandler()))

	err := s.hm.RegisterHandler(test_handler_foreign_arguments.NewHandler())
	s.Error(err)
	s.Equal(err.Error(), fmt.Sprintf(`Handler '%s' version '%s' parameter: Structure must be defined in the same package`, `/test/handler_foreign_arguments`, `V1`))

	err = s.hm.RegisterHandler(test_handler_foreign_return_values.NewHandler())
	s.Error(err)
	s.Equal(err.Error(), fmt.Sprintf(`Handler '%s' version '%s' return value: Structure must be defined in the same package`, `/test/handler_foreign_return_values`, `V1`))
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
	hv1 := s.hm.FindHandler("/test/handler1", 1)
	hv2 := s.hm.FindHandler("/test/handler1", 2)

	s.Equal("v1", hv1.Version)
	s.True(hv1.UseCache)
	s.Equal([]HandlerParameter{
		HandlerParameter{
			Name:        "ReqInt",
			Description: "Required integer argument",
			Key:         "req_int",
			IsRequired:  true,
			RawType:     hv1.Request.Fields[0].RawType,
			getMethod:   hv1.Request.Fields[0].getMethod,
			structField: hv1.Request.Fields[0].structField,
		},
		HandlerParameter{
			Name:        "Int",
			Description: "Unrequired integer argument",
			Key:         "int",
			IsRequired:  false,
			RawType:     hv1.Request.Fields[1].RawType,
			getMethod:   hv1.Request.Fields[1].getMethod,
			structField: hv1.Request.Fields[1].structField,
		},
	}, hv1.Request.Fields)

	s.Equal("v2", hv2.Version)
	s.False(hv2.UseCache)
	s.Equal([]HandlerError{
		HandlerError{
			UserMessage: "Error 1 description",
			Err:         hv2.Errors[0].Err,
			Code:        "ERROR_TYPE1",
		},
		HandlerError{
			UserMessage: "Error 2 description",
			Err:         hv2.Errors[1].Err,
			Code:        "ERROR_TYPE2",
		},
		HandlerError{
			UserMessage: "Error 3 description",
			Err:         hv2.Errors[2].Err,
			Code:        "ERROR_TYPE3",
		},
	}, hv2.Errors)
}

func (s *HandlersManagerSuite) TestHandlerManager_CallHandler1V1_ReturnResult() {
	pg := &ParametersGetter{
		map[string][]string{
			"req_int": []string{"123"},
		},
	}

	hanlerVersion := s.hm.FindHandler("/test/handler1", 1)
	if hanlerVersion == nil {
		s.NotNil(hanlerVersion)
	}

	params, err := s.hm.UnmarshalParameters(context.TODO(), hanlerVersion, pg)
	s.NoError(err)

	res, err := s.hm.CallHandler(context.TODO(), hanlerVersion, params)
	s.NoError(err)
	s.Equal(&test_handler1.V1Res{"Test", 123}, res)
}

func (s *HandlersManagerSuite) TestHandlerManager_PrepareParametersWithError() {
	pg := &ParametersGetter{
		map[string][]string{},
	}

	hanlerVersion := s.hm.FindHandler("/test/handler1", 1)
	if hanlerVersion == nil {
		s.NotNil(hanlerVersion)
	}

	_, err := s.hm.UnmarshalParameters(context.TODO(), hanlerVersion, pg)
	s.Error(err)
}

func TestUnmarshalJsonParameters(t *testing.T) {
	type Request struct {
		IntField          int     `key:"int" json:"int" description:"int field"`
		OptionalIntField  *int    `key:"opt_int" json:"opt_int,omitempty" description:"optional int field"`
		BoolField         bool    `key:"bool" json:"bool" description:"bool field"`
		OptionalBoolField *bool   `key:"opt_bool" json:"opt_bool,omitempty" description:"optional bool field"`
		StrField          string  `key:"str" json:"str" description:"str field"`
		OptionalStrField  *string `key:"opt_str" json:"opt_str,omitempty" description:"optional str field"`
	}
	request := &Request{
		IntField:  1,
		BoolField: true,
		StrField:  "test",
	}
	err := unmarshalJsonParameters(t, request)
	if err != nil {
		t.Fatal(err)
	}
}

func TestUnmarshalOptionalJsonParameters(t *testing.T) {
	type Request struct {
		IntField          int     `key:"int" json:"int" description:"int field"`
		OptionalIntField  *int    `key:"opt_int" json:"opt_int,omitempty" description:"optional int field"`
		BoolField         bool    `key:"bool" json:"bool" description:"bool field"`
		OptionalBoolField *bool   `key:"opt_bool" json:"opt_bool,omitempty" description:"optional bool field"`
		StrField          string  `key:"str" json:"str" description:"str field"`
		OptionalStrField  *string `key:"opt_str" json:"opt_str,omitempty" description:"optional str field"`
	}
	i := 2
	b := true
	s := "optional"
	request := &Request{
		IntField:          1,
		OptionalIntField:  &i,
		BoolField:         true,
		OptionalBoolField: &b,
		StrField:          "test",
		OptionalStrField:  &s,
	}
	err := unmarshalJsonParameters(t, request)
	if err != nil {
		t.Fatal(err)
	}
}

func TestUnmarshalJsonParametersNested(t *testing.T) {
	type Nested1 struct {
		Foo int `key:"int" json:"int" description:"foo field"`
	}
	type Nested2 struct {
		Foo *int `key:"int" json:"int" description:"optional foo field"`
	}
	type Request struct {
		NestedField         Nested1             `key:"nested" json:"nested" description:"nested field"`
		OptionalNestedField *Nested2            `key:"opt_nested" json:"opt_nested,omitempty" description:"nested field"`
		NestedSlice         []Nested1           `key:"nested_slice" json:"nested_slice,omitempty" description:"nested slice"`
		OptionalNestedSlice *[]Nested1          `key:"opt_nested_slice" json:"opt_nested_slice,omitempty" description:"nested slice"`
		NestedMap           map[string]Nested1  `key:"nested_map" json:"nested_map" description:"nested map"`
		OptionalNestedMap   *map[string]Nested1 `key:"opt_nested_map" json:"opt_nested_map,omitempty" description:"nested map"`
	}
	i := 1
	request := &Request{
		NestedField: Nested1{
			Foo: 1,
		},
		OptionalNestedField: &Nested2{
			Foo: &i,
		},
		NestedSlice: []Nested1{Nested1{1}, Nested1{1}},
		NestedMap: map[string]Nested1{
			"1": Nested1{1},
			"2": Nested1{2},
		},
	}
	err := unmarshalJsonParameters(t, request)
	if err != nil {
		t.Fatal(err)
	}
}

func unmarshalJsonParameters(t *testing.T, request interface{}) error {
	handlerRequest, err := processRequestType(reflect.TypeOf(request))
	if err != nil {
		return err
	}

	body, err := json.Marshal(request)
	if err != nil {
		return err
	}
	requestValue, err := unmarshalRequest(handlerRequest, &JsonParametersGetter{
		Req: string(body),
	})
	if err != nil {
		return err
	}

	if !reflect.DeepEqual(request, requestValue.Interface()) {
		return fmt.Errorf("Unmarshalled request is not equal to expected.\nExpected:%+v\nActual:%+v", request, requestValue.Interface())
	}
	return nil
}
