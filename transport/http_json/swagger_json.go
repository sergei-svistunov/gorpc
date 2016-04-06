package http_json

import (
	"bytes"
	"reflect"
	"strings"

	"github.com/sergei-svistunov/gorpc"
)

type Swagger struct {
	SpecVersion         string              `json:"swagger"`
	Info                Info                `json:"info"`
	BasePath            string              `json:"basePath"`
	Host                string              `json:"host,omitempty"`
	Schemes             []string            `json:"schemes,omitempty"`
	Consumes            []string            `json:"consumes,omitempty"`
	Produces            []string            `json:"produces,omitempty"`
	Paths               map[string]PathItem `json:"paths"`
	Tags                []Tag               `json:"tags,omitempty"`
	Definitions         Definitions         `json:"definitions,omitempty"`
	SecurityDefinitions SecurityDefinitions `json:"securityDefinitions,omitempty"`
}

type Info struct {
	Version     string `json:"version"`
	Title       string `json:"title"`
	Description string `json:"description"`
}

type PathItem map[string]*Operation

type Operation struct {
	Tags        []string               `json:"tags,omitempty"`
	Summary     string                 `json:"summary"`
	Description string                 `json:"description"`
	Consumes    []string               `json:"consumes,omitempty"`
	Produces    []string               `json:"produces,omitempty"`
	Parameters  []*Parameter           `json:"parameters,omitempty"`
	Responses   Responses              `json:"responses,omitempty"`
	Security    []*SecurityRequirement `json:"security,omitempty"`
	ExtraData   interface{}            `json:"-"`
}

type Parameter struct {
	Schema
	// used for body parameter (in == "body")
	BodySchema       *Schema `json:"schema,omitempty"`
	Name             string  `json:"name"`
	In               string  `json:"in"`
	Description      string  `json:"description"`
	Required         bool    `json:"required"`
	CollectionFormat string  `json:"collectionFormat,omitempty"`
}

type Tag struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
}

type Items struct {
	Schema
}

type Responses map[string]*Response

type Response struct {
	Description string  `json:"description"`
	Schema      *Schema `json:"schema"`
}

type Schema struct {
	Ref                  string     `json:"$ref,omitempty"`
	Type                 string     `json:"type,omitempty"`
	Description          string     `json:"description,omitempty"`
	Required             []string   `json:"required,omitempty"`
	Items                *Items     `json:"items,omitempty"`
	Properties           Properties `json:"properties,omitempty"`
	AdditionalProperties *Schema    `json:"additionalProperties,omitempty"`
}

type Properties map[string]*Schema

type Definitions map[string]interface{}

// SecurityRequirement security requirement
type SecurityRequirement map[string][]string

// SecurityDefinitions security definitions
type SecurityDefinitions map[string]*SecurityScheme

// SecurityScheme security scheme
type SecurityScheme struct {
	Type        string `json:"type"`
	Description string `json:"description,omitempty"`
	Name        string `json:"name"`
	In          string `json:"in"`
}

// SwaggerJSONCallbacks is struct for callbacks describing
type SwaggerJSONCallbacks struct {
	OnPrepareBaseInfoJSON func(info *Info)
	OnPrepareHandlerJSON  func(path string, data *Operation)
	Process               func(swagger *Swagger)
	TagName               func(path string) string
}

func GenerateSwaggerJSON(hm *gorpc.HandlersManager, host string, callbacks SwaggerJSONCallbacks) (*Swagger, error) {
	swagger := &Swagger{
		SpecVersion: "2.0",
		Info: Info{
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
			<h3>Response compression and caching</h3>
			<p>API compress a response using gzip if the header "Accept-Encoding" contains "gzip" and a response is bigger or equal 1Kb.
			If a response is compressed then server sends the header "Content-Encoding: gzip".</p>
			<p>API supports ETag.</p>`,
		},
		BasePath:    "/",
		Host:        host,
		Consumes:    []string{"application/json"},
		Produces:    []string{"application/json"},
		Paths:       map[string]PathItem{},
		Definitions: Definitions{},
	}

	if callbacks.OnPrepareBaseInfoJSON != nil {
		callbacks.OnPrepareBaseInfoJSON(&swagger.Info)
	}

	for _, path := range hm.GetHandlersPaths() {
		var tagName string

		info := hm.GetHandlerInfo(path)

		if callbacks.TagName == nil {
			tagName = strings.Split(path, "/")[1]
		} else {
			tagName = callbacks.TagName(path)
		}

		swagger.Tags = append(swagger.Tags, Tag{Name: tagName})

		for _, v := range info.Versions {
			operation := &Operation{
				Summary:     info.Caption,
				Description: info.Description,
				Produces:    []string{"application/json"},
				Tags:        []string{tagName},
				ExtraData:   v.ExtraData,
			}

			if !v.Request.Flat {
				bodySchema := getOrCreateSchema(swagger.Definitions, v.Request.Type)
				param := &Parameter{
					Name:        "body",
					Description: "Body",
					In:          "body",
					Required:    true,
					BodySchema:  bodySchema,
				}
				operation.Consumes = []string{"application/json"}
				operation.Parameters = append(operation.Parameters, param)

			} else {
				for _, p := range v.Request.Fields {
					paramType := typeName(p.RawType)
					var arrayType string
					if paramType == "array" {
						arrayType = typeName(p.RawType.Elem())
					}

					param := &Parameter{
						Name:        p.GetKey(),
						Description: p.Description,
						In:          "query",
						Required:    p.IsRequired,
						Schema:      Schema{Type: paramType},
					}
					if arrayType != "" {
						param.CollectionFormat = "multi"
						param.Items = &Items{Schema{Type: arrayType}}
					}
					operation.Parameters = append(operation.Parameters, param)
				}
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
				operation.Responses = Responses{
					"200": &Response{
						Description: "Successful result",
						Schema:      getOrCreateSchema(swagger.Definitions, v.Response),
					},
				}
			}

			if callbacks.OnPrepareHandlerJSON != nil {
				callbacks.OnPrepareHandlerJSON(path, operation)
			}

			var method string
			if v.Request.Flat {
				method = "get"
			} else {
				method = "post"
			}
			swagger.Paths[v.Route] = PathItem{
				method: operation,
			}
		}
	}

	if callbacks.Process != nil {
		callbacks.Process(swagger)
	}
	return swagger, nil
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

func getOrCreateSchema(definitions Definitions, t reflect.Type) *Schema {
	var result Schema
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}

	if t.Kind() == reflect.Map {
		if t.Key().Kind() != reflect.String {
			panic("swagger supports only maps with string keys")
		}
		result.Type = "object"
		result.AdditionalProperties = getOrCreateSchema(definitions, t.Elem())
		return &result
	}

	if t.Kind() == reflect.Interface {
		result.Type = "object"
		return &result
	}

	result.Type = typeName(t)
	if result.Type == "object" {
		name := t.PkgPath() + "/" + t.String()
		if _, ok := definitions[name]; ok {
			result = Schema{Ref: "#/definitions/" + name}
			return &result
		}
		definitions[name] = result

		if t.NumField() > 0 {
			result.Properties = Properties{}
		}
		for i := 0; i < t.NumField(); i++ {
			field := t.Field(i)
			if field.PkgPath != "" {
				continue
			}
			name := field.Tag.Get("json")
			if name == "" {
				name = field.Tag.Get("key")
				if name == "" {
					name = field.Name
				}
			}
			if field.Type.Kind() != reflect.Ptr {
				result.Required = append(result.Required, name)
			}
			fieldSchema := getOrCreateSchema(definitions, field.Type)
			fieldSchema.Description = field.Tag.Get("description")
			result.Properties[name] = fieldSchema
		}
		definitions[name] = result
		result = Schema{Ref: "#/definitions/" + name}
	} else if result.Type == "array" {
		itemsSchema := getOrCreateSchema(definitions, t.Elem())
		result.Items = &Items{*itemsSchema}
	}

	return &result
}
