package adapter

import (
	"bytes"
	"go/format"
	"log"
	"reflect"
	"regexp"
	"strings"

	"fmt"
	"github.com/sergei-svistunov/gorpc"
)

type handlerInfo struct {
	Params []gorpc.HandlerParameter `json:"params"`
	Output string                   `json:"output"`
	Input  string                   `json:"input"`
}

type HttpJsonLibGenerator struct {
	hm                      *gorpc.HandlersManager
	pkgName                 string
	serviceName             string
	path2HandlerInfoMapping map[string]handlerInfo
	collectedStructs        map[string]struct{}
	extraImports            map[string]struct{}
	convertedStructs        map[reflect.Type]string
}

func NewHttpJsonLibGenerator(hm *gorpc.HandlersManager, packageName, serviceName string) *HttpJsonLibGenerator {
	generator := HttpJsonLibGenerator{
		hm:                      hm,
		pkgName:                 "adapter",
		serviceName:             "ExternalAPI",
		path2HandlerInfoMapping: map[string]handlerInfo{},
		collectedStructs:        map[string]struct{}{},
		extraImports:            map[string]struct{}{},
		convertedStructs:        map[reflect.Type]string{},
	}
	if packageName != "" {
		generator.pkgName = packageName
	}
	if serviceName != "" {
		generator.serviceName = serviceName
	}

	return &generator
}

func (g *HttpJsonLibGenerator) Generate() ([]byte, error) {
	structsBuf := &bytes.Buffer{}

	if err := g.collectStructs(structsBuf); err != nil {
		return nil, err
	}

	result := regexp.MustCompilePOSIX(">>>API_NAME<<<").ReplaceAll(mainTemplate, []byte(strings.Title(g.serviceName)))
	result = regexp.MustCompilePOSIX(">>>PKG_NAME<<<").ReplaceAll(result, []byte(g.pkgName))
	result = regexp.MustCompilePOSIX(">>>CLIENT_API_FUNCS<<<").ReplaceAll(result, g.generateAdapterMethods())
	result = regexp.MustCompilePOSIX(">>>CLIENT_STRUCTS<<<").ReplaceAll(result, structsBuf.Bytes())
	result = regexp.MustCompilePOSIX(">>>IMPORTS<<<").ReplaceAll(result, []byte(g.collectImports()))

	return format.Source(result)
}

func (g *HttpJsonLibGenerator) collectStructs(structsBuf *bytes.Buffer) error {
	for _, path := range g.hm.GetHandlersPaths() {
		info := g.hm.GetHandlerInfo(path)
		for _, v := range info.Versions {
			handlerOutputTypeName, err := g.convertStructToCode(v.Response, structsBuf)
			if err != nil {
				return err
			}
			handlerInputTypeName, err := g.convertStructToCode(v.Request.Type, structsBuf)
			if err != nil {
				return err
			}
			g.path2HandlerInfoMapping[path+"/"+v.Version] = handlerInfo{
				Params: v.Request.Fields,
				Output: handlerOutputTypeName,
				Input:  handlerInputTypeName,
			}
		}
	}
	return nil
}

func (g *HttpJsonLibGenerator) generateAdapterMethods() []byte {
	var result bytes.Buffer

	for path, handlerInfo := range g.path2HandlerInfoMapping {
		name := strings.Replace(strings.Title(path), "/", "", -1)
		name = strings.Replace(name, "_", "", -1)

		method := regexp.MustCompilePOSIX(">>>HANDLER_PATH<<<").ReplaceAll(handlerCallPostFuncTemplate, []byte(path))
		method = regexp.MustCompilePOSIX(">>>HANDLER_NAME<<<").ReplaceAll(method, []byte(name))
		method = regexp.MustCompilePOSIX(">>>INPUT_TYPE<<<").ReplaceAll(method, []byte(handlerInfo.Input))
		method = regexp.MustCompilePOSIX(">>>RETURNED_TYPE<<<").ReplaceAll(method, []byte(handlerInfo.Output))
		method = regexp.MustCompilePOSIX(">>>API_NAME<<<").ReplaceAll(method, []byte(strings.Title(g.serviceName)))

		result.Write(method)
	}

	return result.Bytes()
}

func (g *HttpJsonLibGenerator) needToMigratePkgStructs(pkgPath string) bool {
	// TODO this check was removed and all types with non-empty package path will be migrated in library code
	return pkgPath != ""
}

func (g *HttpJsonLibGenerator) convertStructToCode(t reflect.Type, codeBuf *bytes.Buffer) (typeName string, err error) {
	if name, ok := g.convertedStructs[t]; ok {
		return name, nil
	}
	defer func() {
		g.convertedStructs[t] = typeName
	}()

	// ignore slice of new types because this type exactly new and we're collecting its content right now below
	typeName, _ = g.detectTypeName(t)
	if strings.Contains(typeName, ".") {
		// do not migrate external structs (type name with path)
		return
	}

	var newInternalTypes []reflect.Type

	defer func() {
		for _, newType := range newInternalTypes {
			if _, err = g.convertStructToCode(newType, codeBuf); err != nil {
				return
			}
		}
	}()

	switch t.Kind() {
	case reflect.Struct:
		str := "type " + typeName + " struct {\n"
		for i := 0; i < t.NumField(); i++ {
			field := t.Field(i)

			fieldName, emb := g.detectTypeName(field.Type)
			if emb != nil {
				newInternalTypes = append(newInternalTypes, emb...)
			}

			if field.Anonymous {
				str += ("	" + fieldName)
			} else {
				str += ("	" + field.Name + " " + fieldName)
				jsonTag := field.Tag.Get("json")
				if jsonTag == "" {
					jsonTag = field.Tag.Get("key")
				}
				if jsonTag != "" {
					str += (" `json:\"" + jsonTag + "\"`")
				}
			}

			str += "\n"
		}
		str += "}\n\n"

		codeBuf.WriteString(str)

		return
	case reflect.Ptr:
		return g.convertStructToCode(t.Elem(), codeBuf)
	case reflect.Slice:
		var elemType string
		elemType, err = g.convertStructToCode(t.Elem(), codeBuf)
		if err != nil {
			return
		}
		sliceType := "[]" + elemType
		if typeName != sliceType {
			writeType(codeBuf, typeName, sliceType)
		}

		return
	case reflect.Map:
		keyType, _ := g.convertStructToCode(t.Key(), codeBuf)
		valType, _ := g.convertStructToCode(t.Elem(), codeBuf)

		mapName := "map[" + keyType + "]" + valType
		if typeName != mapName {
			writeType(codeBuf, typeName, mapName)
		}
		return
	default:
		// if type is custom we need to describe it in code
		if typeName != t.Kind().String() && typeName != "interface{}" {
			writeType(codeBuf, typeName, t.Kind().String())
			return
		}
	}

	return
}

func writeType(codeBuf *bytes.Buffer, name, kind string) {
	fmt.Fprintf(codeBuf, "type %s %s\n\n", name, kind)
}

func (g *HttpJsonLibGenerator) migratedStructName(t reflect.Type) string {
	path := t.PkgPath()
	if strings.HasPrefix(path, g.hm.Pkg()) {
		path = strings.TrimPrefix(path, g.hm.Pkg())
	}
	if strings.HasPrefix(path, "/") {
		path = strings.TrimPrefix(path, "/")
	}
	path = strings.Replace(path, "/", " ", -1)
	path = strings.Replace(path, ".", "", -1)
	name := strings.Title(path + " " + t.Name())
	name = strings.Replace(name, " ", "", -1)
	return name
}

func (g *HttpJsonLibGenerator) detectTypeName(t reflect.Type) (name string, newTypes []reflect.Type) {
	name = t.Name()
	if name != "" {
		// for custom types make unique names using package path
		// because different packages can contains structs with same names
		if g.needToMigratePkgStructs(t.PkgPath()) {
			name = g.migratedStructName(t)

			if _, exists := g.collectedStructs[name]; !exists {
				newTypes = []reflect.Type{t}
				g.collectedStructs[name] = struct{}{}
			}
		} else if t.PkgPath() != "" {
			g.extraImports[t.PkgPath()] = struct{}{}
			name = t.String()
		}

		name = strings.Replace(name, "-", "_", -1)

		return name, newTypes
	}

	// some types has no name so we need to make it manually
	name = "interface{}"
	switch t.Kind() {
	case reflect.Slice:
		name, embedded := g.detectTypeName(t.Elem())
		if embedded != nil {
			newTypes = append(newTypes, embedded...)
		}
		return "[]" + name, newTypes
	case reflect.Map:
		// TODO enhance for custom key type in map
		key := t.Key().Name()
		val, embedded := g.detectTypeName(t.Elem())
		if embedded != nil {
			newTypes = append(newTypes, embedded...)
		}
		if g.needToMigratePkgStructs(t.Elem().PkgPath()) {
			if _, exists := g.collectedStructs[name]; !exists {
				newTypes = []reflect.Type{t}
				g.collectedStructs[name] = struct{}{}
			}
		}
		if key != "" && val != "" {
			return "map[" + key + "]" + val, newTypes
		}
	case reflect.Ptr:
		name, embedded := g.detectTypeName(t.Elem())
		if embedded != nil {
			newTypes = append(newTypes, embedded...)
		}
		return name, newTypes
	case reflect.Interface:
		return
	default:
		log.Println("Unknown type has been replaced with interface{}")
		return
	}

	return
}

func (g *HttpJsonLibGenerator) collectImports() string {
	var buf bytes.Buffer
	for _, _import := range mainImports {
		appendImport(&buf, _import)
	}
	for _import, _ := range g.extraImports {
		appendImport(&buf, _import)
	}
	return buf.String()
}

func appendImport(buf *bytes.Buffer, _import string) {
	if strings.HasSuffix(_import, `"`) {
		fmt.Fprintln(buf, _import)
	} else {
		fmt.Fprintf(buf, "\"%s\"\n", _import)
	}
}
