package adapter

import (
	"bytes"
	"fmt"
	"go/format"
	"io"
	"log"
	"reflect"
	"regexp"
	"strings"
	"unicode"

	"github.com/sergei-svistunov/gorpc"
)

type HttpJsonLibGenerator struct {
	hm               *gorpc.HandlersManager
	pkgName          string
	serviceName      string
	collectedStructs map[string]struct{}
	extraImports     map[string]struct{}
	convertedStructs map[reflect.Type]string
}

func NewHttpJsonLibGenerator(hm *gorpc.HandlersManager, packageName, serviceName string) *HttpJsonLibGenerator {
	generator := HttpJsonLibGenerator{
		hm:               hm,
		pkgName:          "adapter",
		serviceName:      "ExternalAPI",
		collectedStructs: map[string]struct{}{},
		extraImports:     map[string]struct{}{},
		convertedStructs: map[reflect.Type]string{},
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
	clientAPI, err := g.generateAPI()
	if err != nil {
		return nil, err
	}

	result := regexp.MustCompilePOSIX(">>>API_NAME<<<").ReplaceAll(mainTemplate, []byte(GetAPIName(g.serviceName)))
	result = regexp.MustCompilePOSIX(">>>PKG_NAME<<<").ReplaceAll(result, []byte(g.pkgName))
	result = regexp.MustCompilePOSIX(">>>CLIENT_API<<<").ReplaceAll(result, clientAPI)
	result = regexp.MustCompilePOSIX(">>>IMPORTS<<<").ReplaceAll(result, g.collectImports())

	return format.Source(result)
}

// GetAPIName returns titled service name (first uppercase letter)
// and all occurences of '/._-\s' removed and next to them character uppercased.
func GetAPIName(serviceName string) string {
	if serviceName == "" {
		return ""
	}
	specials := []byte(`/._- `)
	var buf bytes.Buffer
	buf.WriteRune(unicode.ToUpper(rune(serviceName[0])))
	for i := 1; i < len(serviceName); i++ {
		r := rune(serviceName[i])
		if bytes.IndexRune(specials, r) != -1 {
			if i == len(serviceName)-1 {
				break
			}
			i++
			r = unicode.ToUpper(rune(serviceName[i]))
		}
		buf.WriteRune(r)
	}
	return buf.String()
}

func (g *HttpJsonLibGenerator) generateAPI() ([]byte, error) {
	var result bytes.Buffer
	var typesBuf bytes.Buffer

	for _, path := range g.hm.GetHandlersPaths() {
		info := g.hm.GetHandlerInfo(path)
		for _, v := range info.Versions {
			inTypeName, outTypeName, err := g.printHandlerInOutTypes(&typesBuf, v.Request.Type, v.Response)
			if err != nil {
				return nil, err
			}
			handlerTypeName := strings.Replace(strings.Title(v.Route), "/", "", -1)
			handlerTypeName = strings.Replace(handlerTypeName, "_", "", -1)

			errVarName := g.printHandlerMethodError(&typesBuf, handlerTypeName, v.Errors)

			method := regexp.MustCompilePOSIX(">>>HANDLER_PATH<<<").ReplaceAll(handlerCallPostFuncTemplate, []byte(v.Route))
			method = regexp.MustCompilePOSIX(">>>HANDLER_NAME<<<").ReplaceAll(method, []byte(handlerTypeName))
			method = regexp.MustCompilePOSIX(">>>INPUT_TYPE<<<").ReplaceAll(method, []byte(inTypeName))
			method = regexp.MustCompilePOSIX(">>>RETURNED_TYPE<<<").ReplaceAll(method, []byte(outTypeName))
			method = regexp.MustCompilePOSIX(">>>HANDLER_ERRORS<<<").ReplaceAll(method, []byte(errVarName))
			method = regexp.MustCompilePOSIX(">>>API_NAME<<<").ReplaceAll(method, []byte(GetAPIName(g.serviceName)))

			result.Write(method)
		}
	}

	typesBuf.WriteTo(&result)

	return result.Bytes(), nil
}

func (g *HttpJsonLibGenerator) printHandlerInOutTypes(w io.Writer, in, out reflect.Type) (inTypeName string, outTypeName string, err error) {
	inTypeName, err = g.convertStructToCode(w, in)
	if err != nil {
		return
	}

	outTypeName, err = g.convertStructToCode(w, out)
	if err != nil {
		return
	}
	if out.Kind() == reflect.Struct || (out.Kind() == reflect.Ptr && out.Elem().Kind() == reflect.Struct) {
		outTypeName = "*" + outTypeName
	}

	return
}

func (g *HttpJsonLibGenerator) printHandlerMethodError(w io.Writer, handlerTypeName string, errors []gorpc.HandlerError) string {
	if len(errors) == 0 {
		return "nil"
	}

	handlerErrorsName := handlerTypeName + "Errors"
	fmt.Fprintf(w, "type %s int\n\n", handlerErrorsName)
	fmt.Fprint(w, "const (\n")
	for i, e := range errors {
		if i == 0 {
			fmt.Fprintf(w, "%s_%s = iota\n", handlerErrorsName, e.Code)
		} else {
			fmt.Fprintf(w, "%s_%s\n", handlerErrorsName, e.Code)
		}
	}
	fmt.Fprint(w, ")\n\n")
	fmt.Fprintf(w, "var _%sMapping = map[string]int{\n", handlerErrorsName)
	for _, e := range errors {
		fmt.Fprintf(w, "\"%s\": %s_%s,\n", e.Code, handlerErrorsName, e.Code)
	}
	fmt.Fprint(w, "}\n\n")
	return "_" + handlerErrorsName + "Mapping"
}

func (g *HttpJsonLibGenerator) needToMigratePkgStructs(pkgPath string) bool {
	// TODO this check was removed and all types with non-empty package path will be migrated in library code
	return pkgPath != ""
}

func (g *HttpJsonLibGenerator) convertStructToCode(w io.Writer, t reflect.Type) (typeName string, err error) {
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
			if _, err = g.convertStructToCode(w, newType); err != nil {
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

		fmt.Fprint(w, str)

		return
	case reflect.Ptr:
		return g.convertStructToCode(w, t.Elem())
	case reflect.Slice:
		var elemType string
		elemType, err = g.convertStructToCode(w, t.Elem())
		if err != nil {
			return
		}
		sliceType := "[]" + elemType
		if typeName != sliceType {
			writeType(w, typeName, sliceType)
		}

		return
	case reflect.Map:
		keyType, _ := g.convertStructToCode(w, t.Key())
		valType, _ := g.convertStructToCode(w, t.Elem())

		mapName := "map[" + keyType + "]" + valType
		if typeName != mapName {
			writeType(w, typeName, mapName)
		}
		return
	default:
		// if type is custom we need to describe it in code
		if typeName != t.Kind().String() && typeName != "interface{}" {
			writeType(w, typeName, t.Kind().String())
			return
		}
	}

	return
}

func writeType(w io.Writer, name, kind string) {
	fmt.Fprintf(w, "type %s %s\n\n", name, kind)
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

func (g *HttpJsonLibGenerator) collectImports() []byte {
	var buf bytes.Buffer
	for _, _import := range mainImports {
		appendImport(&buf, _import)
	}
	for _import, _ := range g.extraImports {
		appendImport(&buf, _import)
	}
	return buf.Bytes()
}

func appendImport(buf *bytes.Buffer, _import string) {
	if strings.HasSuffix(_import, `"`) {
		fmt.Fprintln(buf, _import)
	} else {
		fmt.Fprintf(buf, "\"%s\"\n", _import)
	}
}
