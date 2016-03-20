package jsonrpc

import (
	"encoding/json"
	"net/http"

	"github.com/sergei-svistunov/gorpc"
	"reflect"
)

type SchemeHandler struct {
	schemeJson []byte
}

type SchemeInfo struct {
	URL     string             `json:"url,omitempty"`
	Methods []SchemeMethodInfo `json:"methods,omitempty"`
}

type SchemeMethodInfo struct {
	Name        string                  `json:"name,omitempty"`
	Caption     string                  `json:"caption,omitempty"`
	Description string                  `json:"description,omitempty"`
	Errors      []SchemeMethodInfoError `json:"errors,omitempty"`
	Params      *SchemeMethodInfoParam  `json:"params,omitempty"`
}

type SchemeMethodInfoError struct {
	Id      int    `json:"id,omitempty"`
	Message string `json:"message,omitempty"`
	Data    string `json:"data,omitempty"`
}

type SchemeMethodInfoParam struct {
	Type        string                           `json:"type"`
	Required    bool                             `json:"required,omitempty"`
	Description string                           `json:"description,omitempty"`
	KeyType     string                           `json:"key_type,omitempty"`
	ValueType   *SchemeMethodInfoParam           `json:"value_type,omitempty"`
	Fields      map[string]SchemeMethodInfoParam `json:"fields,omitempty"`
}

func NewSchemeHandler(hm *gorpc.HandlersManager, apiPort uint16) *SchemeHandler {
	schemeInfo := SchemeInfo{
		Methods: make([]SchemeMethodInfo, 0),
	}

	for _, path := range hm.GetHandlersPaths() {
		handlerInfo := hm.GetHandlerInfo(path)

		for _, version := range handlerInfo.Versions {
			methodInfo := SchemeMethodInfo{
				Name:        version.Route,
				Caption:     handlerInfo.Caption,
				Description: handlerInfo.Description,
				Errors:      make([]SchemeMethodInfoError, len(version.Errors)),
			}

			for i, error := range version.Errors {
				methodInfo.Errors[i].Id = -32000 //ToDo: Fix it
				methodInfo.Errors[i].Message = error.Code
				methodInfo.Errors[i].Data = error.UserMessage
			}

			if len(version.Request.Fields) > 0 {
				methodInfo.Params = &SchemeMethodInfoParam{
					Type:   "object",
					Fields: make(map[string]SchemeMethodInfoParam),
				}

				for _, field := range version.Request.Fields {
					methodInfo.Params.Fields[field.GetKey()] = handlerParamToSchemeParam(field.RawType, nil)
				}
			}

			schemeInfo.Methods = append(schemeInfo.Methods, methodInfo)
		}
	}

	h := &SchemeHandler{}

	h.schemeJson, _ = json.Marshal(schemeInfo)

	return h
}

func (h *SchemeHandler) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Write(h.schemeJson)
}

func handlerParamToSchemeParam(param reflect.Type, typesStack []string) SchemeMethodInfoParam {
	res := SchemeMethodInfoParam{
		Required: true,
	}

	if param.Kind() == reflect.Ptr {
		res.Required = false
		param = param.Elem()
	}

	for _, p := range typesStack {
		if p == param.Name() {
			return SchemeMethodInfoParam{
				Type: "recursive",
			}
		}
	}

	switch param.Kind() {
	case reflect.Array, reflect.Slice:
		res.Type = "array"
		valueType := handlerParamToSchemeParam(param.Elem(), append(typesStack, param.Name()))
		res.ValueType = &valueType
	case reflect.Struct:
		res.Type = "object"
		res.Fields = make(map[string]SchemeMethodInfoParam)
		for i := 0; i < param.NumField(); i++ {
			structField := param.Field(i)
			key := structField.Tag.Get("key")
			if key == "" {
				key = structField.Name
			}
			res.Fields[key] = handlerParamToSchemeParam(structField.Type, append(typesStack, structField.Name))
		}
	case reflect.Map:
		res.Type = "map"
		res.KeyType = param.Key().Name()
		valueType := handlerParamToSchemeParam(param.Elem(), append(typesStack, param.Name()))
		res.ValueType = &valueType
	default:
		res.Type = param.Kind().String()
	}

	return res
}
