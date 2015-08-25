package http_json

import (
	"bytes"
	"encoding/json"
	"net/http"
	"reflect"
	"strings"

	"github.com/sergei-svistunov/gorpc"
)

type swagger struct {
	SpecVersion string              `json:"swagger"`
	Info        info                `json:"info"`
	BasePath    string              `json:"basePath"`
	Host        string              `json:"host,omitempty"`
	Schemes     []string            `json:"schemes,omitempty"`
	Consumes    []string            `json:"consumes,omitempty"`
	Produces    []string            `json:"produces,omitempty"`
	Paths       map[string]pathItem `json:"paths"`
	Tags        []tag               `json:"tags,omitempty"`
	Definitions definitions         `json:"definitions,omitempty"`
}

type info struct {
	Version     string `json:"version"`
	Title       string `json:"title"`
	Description string `json:"description"`
}

type pathItem map[string]operation

type operation struct {
	Tags        []string    `json:"tags,omitempty"`
	Summary     string      `json:"summary"`
	Description string      `json:"description"`
	Consumes    []string    `json:"consumes,omitempty"`
	Produces    []string    `json:"produces,omitempty"`
	Parameters  []parameter `json:"parameters,omitempty"`
	Responses   responses   `json:"responses,omitempty"`
}

type parameter struct {
	schema
	Name             string `json:"name"`
	In               string `json:"in"`
	Description      string `json:"description"`
	Required         bool   `json:"required"`
	CollectionFormat string `json:"collectionFormat,omitempty"`
}

type tag struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
}

type items struct {
	schema
}

type responses map[string]response

type response struct {
	Description string `json:"description"`
	Schema      schema `json:"schema"`
}

type schema struct {
	Ref         string                 `json:"$ref,omitempty"`
	Type        string                 `json:"type,omitempty"`
	Description string                 `json:"description,omitempty"`
	Required    []string               `json:"required,omitempty"`
	Items       interface{}            `json:"items,omitempty"`
	Properties  map[string]interface{} `json:"properties,omitempty"`
}

type definitions map[string]interface{}

type SwaggerJSONHandler struct {
	jsonB []byte
}

func NewSwaggerJSONHandler(hm *gorpc.HandlersManager) *SwaggerJSONHandler {
	jsonB, err := generateSwaggerJSON(hm)
	if err != nil {
		panic(err)
	}
	return &SwaggerJSONHandler{
		jsonB: jsonB,
	}
}

func (h *SwaggerJSONHandler) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Write(h.jsonB)
}

func (h *SwaggerJSONHandler) GetJSON() []byte {
	return h.jsonB
}

func generateSwaggerJSON(hm *gorpc.HandlersManager) ([]byte, error) {
	swagger := swagger{
		SpecVersion: "2.0",
		Info: info{
			Version: "1.0.0",
			Title:   "HTTP JSON RPC for Go",
			Description: `<h2>Description</h2>
			<p>HTTPS RPC server.</p>
			<h2>Protocol</h2>
			<p>It supports "GET" or "POST" methods for requests and returns a JSON in response.</p>
			<h3>Response</h3>
			<p>Response is a JSON object that contains 3 fields:
			  <ul>
				<li><strong>result: </strong><code>OK</code>, <code>ERROR</code></li>
				<li><strong>data: </strong>response payload, it is error description if <code>result</code> is <code>ERROR</code></li>
				<li><strong>error: </strong>error code, it is an empty string if <code>result</code> is <code>OK</code></li>
			  </ul>
			</p>
			<h3>Response compression</h3>
			<p>API compress a respone using gzip if the header "Accept-Encoding" contains "gzip" and a response is bigger or equal 1Kb.
			If a response is compressed then server sends the header "Content-Encoding: gzip".</p>`,
		},
		BasePath:    "/",
		Consumes:    []string{"application/json"},
		Produces:    []string{"application/json"},
		Paths:       map[string]pathItem{},
		Definitions: definitions{},
	}

	for _, path := range hm.GetHandlersPaths() {
		info := hm.GetHandlerInfo(path)
		tagName := strings.Split(path, "/")[1]
		swagger.Tags = append(swagger.Tags, tag{Name: tagName})

		for _, v := range info.Versions {
			operation := operation{
				Summary:     info.Caption,
				Description: info.Description,
				Produces:    []string{"application/json"},
				Tags:        []string{tagName},
			}

			if v.UseCache {
				operation.Description += ".<br/>Handler caches response."
			}

			for _, p := range v.Parameters {
				p.Type = typeName(p.RawType)
				var arrayType string
				if p.Type == "array" {
					arrayType = typeName(p.RawType.Elem())
				}

				param := parameter{
					Name:        p.GetKey(),
					Description: p.Description,
					In:          "query",
					Required:    p.IsRequired,
					schema:      schema{Type: p.Type},
				}
				if arrayType != "" {
					param.CollectionFormat = "multi"
					param.Items = items{schema{Type: arrayType}}
				}
				operation.Parameters = append(operation.Parameters, param)
			}

			if len(v.Errors) > 0 {
				var errorsDescription bytes.Buffer
				errorsDescription.WriteString("<br>Handler can return these error messages:\n")
				errorsDescription.WriteString("<ul>")
				for _, e := range v.Errors {
					errorsDescription.WriteString("<li>")
					errorsDescription.WriteString("Code: \"<code>")
					errorsDescription.WriteString(e.Code)
					errorsDescription.WriteString("</code>\", Data: \"<code>")
					errorsDescription.WriteString(e.UserMessage)
					errorsDescription.WriteString("</code>\"</li>")
				}
				errorsDescription.WriteString("</ul>")
				operation.Description += errorsDescription.String()
			}

			if v.Response != nil {
				operation.Responses = responses{
					"200": response{
						Description: "Successful result",
						Schema:      getOrCreateSchema(swagger.Definitions, v.Response),
					},
				}
			}

			swagger.Paths[path+"/"+v.Version+"/"] = pathItem{
				"get": operation,
			}
		}
	}

	return json.Marshal(swagger)
}

func typeName(t reflect.Type) (name string) {
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	switch t.Kind() {
	case reflect.Array, reflect.Slice:
		name = "array"
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		name = "integer"
	case reflect.Float32, reflect.Float64:
		name = "number"
	case reflect.Bool:
		name = "boolean"
	case reflect.String:
		name = "string"
	case reflect.Struct:
		name = "object"
	default:
		panic("unknown type kind " + t.Kind().String())
	}
	return
}

func getOrCreateSchema(definitions definitions, t reflect.Type) schema {
	var result schema
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	// TODO: fix it, interfaces and maps are not supported yet.
	if t.Kind() == reflect.Interface || t.Kind() == reflect.Map {
		result.Type = "object"
		return result
	}

	result.Type = typeName(t)
	if result.Type == "object" {
		name := t.String()
		if _, ok := definitions[name]; ok {
			result = schema{Ref: "#/definitions/" + name}
			return result
		}
		definitions[name] = result

		if t.NumField() > 0 {
			result.Properties = make(map[string]interface{})
		}
		for i := 0; i < t.NumField(); i++ {
			field := t.Field(i)
			if field.PkgPath != "" {
				continue
			}
			name := field.Tag.Get("json")
			if name == "" {
				name = field.Name
			}
			if field.Type.Kind() != reflect.Ptr {
				result.Required = append(result.Required, name)
			}
			fieldSchema := getOrCreateSchema(definitions, field.Type)
			fieldSchema.Description = field.Tag.Get("description")
			result.Properties[name] = fieldSchema
		}
		definitions[name] = result
		result = schema{Ref: "#/definitions/" + name}
	} else if result.Type == "array" {
		itemsSchema := getOrCreateSchema(definitions, t.Elem())
		result.Items = items{itemsSchema}
	}

	return result
}
