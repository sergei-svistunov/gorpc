package adapter

import (
	"bytes"
	"github.com/sergei-svistunov/gorpc"
	"go/format"
	"log"
	"reflect"
	"regexp"
	"strings"
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
	internalPkgs            []string
	path2HandlerInfoMapping map[string]handlerInfo
	collectedStructs        StringsStack
}

func NewHttpJsonLibGenerator(hm *gorpc.HandlersManager, packageName, serviceName string) *HttpJsonLibGenerator {
	generator := HttpJsonLibGenerator{
		hm:                      hm,
		pkgName:                 "adapter",
		serviceName:             "ExternalAPI",
		path2HandlerInfoMapping: map[string]handlerInfo{},
	}
	if packageName != "" {
		generator.pkgName = packageName
	}
	if serviceName != "" {
		generator.serviceName = serviceName
	}

	// TODO collect internal pathes from HM
	generator.internalPkgs = []string{"lazada_api"}

	return &generator
}

func (g *HttpJsonLibGenerator) Generate() ([]byte, error) {
	structsBuf := &bytes.Buffer{}
	extraImports := []string{}

	if err := g.collectStructs(structsBuf, &extraImports); err != nil {
		return nil, err
	}

	result := regexp.MustCompilePOSIX(">>>PKG_NAME<<<").ReplaceAll(mainTemplate, []byte(g.pkgName))
	result = regexp.MustCompilePOSIX(">>>STATIC_LOGIC<<<").ReplaceAll(result, g.getStaticCodeTemplate())
	result = regexp.MustCompilePOSIX(">>>DYNAMIC_LOGIC<<<").ReplaceAll(result, g.generateAdapterMethods(structsBuf))
	result = regexp.MustCompilePOSIX(">>>STRUCTS<<<").ReplaceAll(result, structsBuf.Bytes())
	result = regexp.MustCompilePOSIX(">>>IMPORTS<<<").ReplaceAll(result, []byte(g.collectImports(extraImports)))

	return format.Source(result)

}

func (g *HttpJsonLibGenerator) getStaticCodeTemplate() []byte {
	return regexp.MustCompilePOSIX(">>>API_NAME<<<").ReplaceAll(staticLogicTemplate, []byte(strings.Title(g.serviceName)))
}

func (g *HttpJsonLibGenerator) collectStructs(structsBuf *bytes.Buffer, extraImports *[]string) error {
	for _, path := range g.hm.GetHandlersPaths() {
		info := g.hm.GetHandlerInfo(path)
		for _, v := range info.Versions {
			handlerOutputTypeName, err := g.convertStructToCode(v.GetMethod().Type.Out(0), structsBuf, extraImports)
			if err != nil {
				return err
			}
			handlerIntputTypeName, err := g.convertStructToCode(v.GetMethod().Type.In(2), structsBuf, extraImports)
			if err != nil {
				return err
			}
			g.path2HandlerInfoMapping[path+"/"+v.GetVersion()] = handlerInfo{
				Params: v.Request.Fields,
				Output: handlerOutputTypeName,
				Input:  handlerIntputTypeName,
			}
		}
	}
	return nil
}

func (g *HttpJsonLibGenerator) generateAdapterMethods(structsBuf *bytes.Buffer) []byte {
	result := &bytes.Buffer{}

	for path, handlerInfo := range g.path2HandlerInfoMapping {
		var method []byte
		method = regexp.MustCompilePOSIX(">>>HANDLER_PATH<<<").ReplaceAll(handlerCallPostFuncTemplate, []byte(path))
		path = strings.Replace(strings.Title(path), "/", "", -1)
		method = regexp.MustCompilePOSIX(">>>HANDLER_NAME<<<").ReplaceAll(method, []byte(path))
		method = regexp.MustCompilePOSIX(">>>INPUT_TYPE<<<").ReplaceAll(method, []byte(handlerInfo.Input))
		method = regexp.MustCompilePOSIX(">>>RETURNED_TYPE<<<").ReplaceAll(method, []byte(handlerInfo.Output))
		method = regexp.MustCompilePOSIX(">>>API_NAME<<<").ReplaceAll(method, []byte(strings.Title(g.serviceName)))

		result.Write(method)
	}

	return result.Bytes()
}

func (g *HttpJsonLibGenerator) isInternalType(pkgPath string) bool {
	for _, pkgName := range g.internalPkgs {
		if strings.HasPrefix(pkgPath, pkgName) {
			return true
		}
	}
	return false
}

func (g *HttpJsonLibGenerator) convertStructToCode(t reflect.Type, codeBuf *bytes.Buffer, extraImports *[]string) (typeName string, err error) {
	// ignore slice of new types because this type exactly new and we're collecting its content right now below
	typeName, _ = g.detectTypeName(t, extraImports)
	if strings.Contains(typeName, ".") {
		// do not migrate external structs (type name with path)
		return
	}

	var newInternalTypes []reflect.Type

	defer func() {
		for _, newType := range newInternalTypes {
			if _, err = g.convertStructToCode(newType, codeBuf, extraImports); err != nil {
				return
			}
		}
	}()

	switch t.Kind() {
	case reflect.Struct:
		str := "type " + typeName + " struct {\n"
		for i := 0; i < t.NumField(); i++ {

			field := t.Field(i)

			fieldName, emb := g.detectTypeName(field.Type, extraImports)
			if emb != nil {
				newInternalTypes = append(newInternalTypes, emb...)
			}

			if field.Anonymous {
				str += ("	" + fieldName)
			} else {
				str += ("	" + field.Name + " " + fieldName)
				if jsonTag := field.Tag.Get("json"); jsonTag != "" {
					str += (" `json:\"" + jsonTag + "\"`")
				}
			}

			str += "\n"
		}
		str += "}\n\n"

		codeBuf.WriteString(str)

		return
	case reflect.Ptr:
		return g.convertStructToCode(t.Elem(), codeBuf, extraImports)
	case reflect.Slice:
		var elemType string
		elemType, err = g.convertStructToCode(t.Elem(), codeBuf, extraImports)
		if err != nil {
			return
		}
		sliceType := "[]" + elemType
		if typeName != sliceType {
			writeType(codeBuf, typeName, sliceType)
		}

		return
	case reflect.Map:
		keyType, _ := g.convertStructToCode(t.Key(), codeBuf, extraImports)
		valType, _ := g.convertStructToCode(t.Elem(), codeBuf, extraImports)

		if mapName := "map[" + keyType + "]" + valType; typeName != mapName {
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
	codeBuf.WriteString("type " + name + " " + kind + "\n\n")
}

func (g *HttpJsonLibGenerator) detectTypeName(t reflect.Type, extraImports *[]string) (name string, newTypes []reflect.Type) {
	name = t.Name()
	if name != "" {
		// for custom types make unique names using package path
		// because different packages can contains structs with same names
		if g.isInternalType(t.PkgPath()) {
			path := strings.Replace(t.PkgPath(), "/", "_", -1)
			path = strings.Title(path)

			name = path + "_" + strings.Title(name)

			if !g.collectedStructs.AlreadyExist(name) {
				newTypes = []reflect.Type{t}
				g.collectedStructs.Add(name)
			}
		} else if t.PkgPath() != "" {
			*extraImports = append(*extraImports, t.PkgPath())
			name = t.String()
		}

		name = strings.Replace(name, "-", "_", -1)

		return name, newTypes
	}

	// some types has no name so we need to make it manually
	name = "interface{}"
	switch t.Kind() {
	case reflect.Slice:
		name, embeded := g.detectTypeName(t.Elem(), extraImports)
		if embeded != nil {
			newTypes = append(newTypes, embeded...)
		}
		return "[]" + name, newTypes
	case reflect.Map:
		// TODO enhance for custom key type in map
		key := t.Key().Name()
		val, embeded := g.detectTypeName(t.Elem(), extraImports)
		if embeded != nil {
			newTypes = append(newTypes, embeded...)
		}
		if g.isInternalType(t.Elem().PkgPath()) {
			if !g.collectedStructs.AlreadyExist(name) {
				newTypes = []reflect.Type{t}
				g.collectedStructs.Add(name)
			}
		}
		if key != "" && val != "" {
			return "map[" + key + "]" + val, newTypes
		}
	case reflect.Ptr:
		name, embeded := g.detectTypeName(t.Elem(), extraImports)
		if embeded != nil {
			newTypes = append(newTypes, embeded...)
		}
		return name, newTypes
	case reflect.Interface:
		return
	}

	log.Println("Unknown type has been replaced with interface{}")
	return
}

func (g *HttpJsonLibGenerator) collectImports(extraImports []string) string {
	imports := mainImports
	if len(extraImports) > 0 {
		imports = append(imports, extraImports...)
	}

	var result string
	for i := range imports {
		if strings.HasSuffix(imports[i], "\"") {
			result += (imports[i] + "\n")
		} else {
			result += ("\"" + imports[i] + "\"\n")
		}
	}

	return result
}
