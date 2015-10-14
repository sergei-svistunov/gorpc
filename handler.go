package gorpc

import (
	"reflect"
)

type handlerVersion struct {
	Parameters    []handlerParameter
	Errors        []HandlerError
	Response      reflect.Type
	Version       string
	UseCache      bool
	AcceptJSON    bool
	ExtraData     interface{}
	path          string
	handlerStruct IHandler
	method        reflect.Method
}

type handlerParameter struct {
	Name        string
	Description string
	Key         string
	RawType     reflect.Type
	IsRequired  bool
	getMethod   reflect.Method
	structField reflect.StructField
}

func (p *handlerParameter) GetKey() string {
	if p.Key != "" {
		return p.Key
	}

	return p.Name
}
