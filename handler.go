package gorpc

import (
	"reflect"
)

type handlerVersion struct {
	Errors        []HandlerError
	Request       *handlerRequest
	Response      reflect.Type
	Version       string
	UseCache      bool
	ExtraData     interface{}
	handlerStruct IHandler
	method        reflect.Method
	path          string
}

type handlerRequest struct {
	Type   reflect.Type
	Flat   bool
	Fields []handlerParameter
}

type handlerParameter struct {
	Name        string
	Description string
	Key         string
	Path        []string
	RawType     reflect.Type
	IsRequired  bool
	getMethod   reflect.Method
	structField reflect.StructField
	Fields      []handlerParameter
}

func (p *handlerParameter) GetKey() string {
	if p.Key != "" {
		return p.Key
	}
	return p.Name
}
