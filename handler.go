package gorpc

import (
	"reflect"
)

type handlerVersion struct {
	Parameters    []HandlerParameter
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
	Fields []HandlerParameter
}

func (v *handlerVersion) GetMethod() reflect.Method {
	return v.method
}

func (v *handlerVersion) GetVersion() string {
	return v.Version
}

type HandlerParameter struct {
	Name        string
	Description string
	Key         string
	Path        []string
	RawType     reflect.Type
	IsRequired  bool
	getMethod   reflect.Method
	structField reflect.StructField
	Fields      []HandlerParameter
}

func (p *HandlerParameter) GetKey() string {
	if p.Key != "" {
		return p.Key
	}
	return p.Name
}
