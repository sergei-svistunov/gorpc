package gorpc

import (
	"reflect"
)

type IHandler interface {
	Caption() string
	Description() string
}

type handlerVersion struct {
	Parameters    []handlerParameter
	Errors        []HandlerError
	Response      reflect.Type
	Version       string
	UseCache      bool
	ExtraData     interface{}
	path          string
	handlerStruct IHandler
	method        reflect.Method
}

type handlerParameter struct {
	Name        string
	Type        string
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
