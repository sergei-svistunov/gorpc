package gorpc

import (
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode"

	"golang.org/x/net/context"
)

type handlerEntity struct {
	path          string
	versions      []handlerVersion
	handlerStruct IHandler
}

type HandlersManager struct {
	handlers     map[string]*handlerEntity
	handlersPath string
	cache        ICache
	cacheTTL     time.Duration
}

func NewHandlersManager(handlersPath string, cache ICache, cacheTTL time.Duration) *HandlersManager {
	return &HandlersManager{
		handlers:     make(map[string]*handlerEntity),
		handlersPath: strings.TrimSuffix(handlersPath, "/"),
		cache:        cache,
		cacheTTL:     cacheTTL,
	}
}

func (hm *HandlersManager) RegisterHandler(h IHandler) error {
	handlerType := reflect.TypeOf(h)
	for handlerType.Kind() != reflect.Ptr {
		return fmt.Errorf("Handler type must be Ptr")
	}
	handlerPtrType := handlerType.Elem()

	if handlerPtrType.Kind() != reflect.Struct {
		return fmt.Errorf("Invalid handler type \"%s\"", handlerPtrType.Name())
	}

	handlerPath := handlerPtrType.PkgPath()
	if strings.HasPrefix(handlerPath, hm.handlersPath+"/") {
		handlerPath = strings.TrimPrefix(handlerPath, hm.handlersPath)
	} else {
		return fmt.Errorf("Handler \"%s\" is in invalid path \"%s\"", handlerPtrType.Name(), handlerPath)
	}

	if _, m := hm.handlers[handlerPath]; m {
		return fmt.Errorf("Handler with path \"%s\" is already exists", handlerPath)
	}

	var handlerVersionsIds []int
	for i := 0; i < handlerType.NumMethod(); i++ {
		name := handlerType.Method(i).Name
		ok, err := regexp.MatchString("^V[1-9]\\d*$", name)
		if err != nil {
			panic(err)
		}
		if ok {
			v, _ := strconv.Atoi(name[1:])
			handlerVersionsIds = append(handlerVersionsIds, v)
		}
	}
	sort.Sort(sort.IntSlice(handlerVersionsIds))
	versions := make([]handlerVersion, len(handlerVersionsIds))

	v := new(IHandlerParameters)
	existingHandlerMethods := reflect.TypeOf(v).Elem()

	for i, v := range handlerVersionsIds {
		handlerVersion := i + 1
		if handlerVersion != v {
			return fmt.Errorf("You have missed version number %d of handler %s", handlerVersion, handlerPath)
		}

		vMethodType, _ := handlerType.MethodByName("V" + strconv.Itoa(v))
		numIn := vMethodType.Type.NumIn()
		if numIn != 3 && numIn != 4 {
			return fmt.Errorf("Invalid prototype for version number %d of handler %s", handlerVersion, handlerPath)
		}

		ctxType := vMethodType.Type.In(1)

		if ctxType.Kind() != reflect.Interface || ctxType.PkgPath() != "golang.org/x/net/context" || ctxType.Name() != "Context" {
			return fmt.Errorf("Invalid prototype for version number %d of handler %s. First argument must be \"Context\" from package \"golang.org/x/net/context\"", handlerVersion, handlerPath)
		}

		paramsType := vMethodType.Type.In(2)
		if paramsType.Kind() != reflect.Ptr || paramsType.Elem().Kind() != reflect.Struct {
			return fmt.Errorf("Type of opts for version number %d of handler %s must be Ptr to Struct", handlerVersion, handlerPath)
		}

		version := &versions[i]
		version.Version = "v" + strconv.Itoa(v)
		version.Parameters = make([]handlerParameter, paramsType.Elem().NumField())
		version.path = handlerPath
		version.method = vMethodType
		version.handlerStruct = h

		if _, ok := handlerType.MethodByName("V" + strconv.Itoa(v) + "UseCache"); ok {
			version.UseCache = true
		}

		if vMethodType.Type.NumOut() != 2 {
			return &CallHandlerError{ErrorInParameters, fmt.Errorf("Invalid count of output parameters for version number %d of handler %s", handlerVersion, handlerPath)}
		}

		if vMethodType.Type.Out(1).String() != "error" {
			return &CallHandlerError{ErrorInParameters, fmt.Errorf("Second output parameter should be error (handler %s version number %d)", handlerPath, handlerVersion)}
		}

		// TODO: check response object for unexported fields here. Move that code out of docs.go
		version.Response = vMethodType.Type.Out(0)

		for pN, parameter := range version.Parameters {
			fieldType := paramsType.Elem().Field(pN)

			parameter.Key = fieldType.Tag.Get("key")
			if parameter.Key == "" || parameter.Key == "-" {
				return fmt.Errorf("tag \"key\" must be specified for parameter %q (handler %s, version number %d)", fieldType.Name, handlerPath, handlerVersion)
			}

			parameter.Name = fieldType.Name
			if unicode.IsLower(rune(fieldType.Name[0])) {
				return fmt.Errorf("Parameters field %s is private (handler %s, version number %d)", parameter.Name, handlerPath, handlerVersion)
			}

			parameter.RawType = fieldType.Type
			if fieldType.Type.Kind() == reflect.Ptr {
				parameter.Type = fieldType.Type.Elem().Kind().String()
			} else {
				parameter.Type = fieldType.Type.Kind().String()
			}

			paramGetMethod, exist := findParameterGetMethod(existingHandlerMethods, fieldType.Type)
			if !exist {
				return fmt.Errorf("Type %s does not supported by handler %s for version number %d", parameter.Type, handlerPath, handlerVersion)
			}
			parameter.getMethod = paramGetMethod
			parameter.structField = fieldType

			parameter.Description = fieldType.Tag.Get("description")
			if parameter.Description == "" {
				return fmt.Errorf("Opt %s of handler %s for version number %d does not have description", fieldType.Name, handlerPath, handlerVersion)
			}
			parameter.IsRequired = fieldType.Type.Kind() != reflect.Ptr

			version.Parameters[pN] = parameter
		}

		// check and prepare errors types for handler
		errMethod, found := handlerType.MethodByName("V" + strconv.Itoa(v) + "ErrorsVar")
		if found {
			errMethodType := errMethod.Type
			if errMethodType.NumOut() == 0 {
				return fmt.Errorf("V%dErrors() method of handler %s should return errors types struct", handlerVersion, handlerPath)
			}
			retValues := errMethod.Func.Call([]reflect.Value{reflect.ValueOf(h)})
			errVar := retValues[0].Elem()

			for i := 0; i < errVar.NumField(); i++ {
				fieldVal := errVar.Field(i)
				fieldStruct := errVar.Type().Field(i)
				if !fieldVal.IsValid() || !fieldVal.CanSet() {
					return fmt.Errorf("Can't set value for ErrorTypes field %s. Handler %s", fieldStruct.Name, handlerPath)
				}

				errText := fieldStruct.Tag.Get("text")
				if errText == "" {
					return fmt.Errorf("ErrorTypes struct is invalid: field '%s' has not any error text. Handler %s", fieldStruct.Name, handlerPath)
				}

				handlerError := HandlerError{
					UserMessage: errText,
					Err:         errors.New(errText),
					Code:        fieldStruct.Name,
				}
				version.Errors = append(version.Errors, handlerError)
				fieldVal.Set(
					reflect.ValueOf(&handlerError),
				)
			}
		}
	}

	hm.handlers[handlerPath] = &handlerEntity{
		path:          handlerPath,
		versions:      versions,
		handlerStruct: h,
	}

	return nil
}

func (hm *HandlersManager) FindHandler(path string, version int) *handlerVersion {
	handler := hm.getHandlerByPath(path)
	if handler == nil {
		return nil
	}

	if version < 1 || version > len(handler.versions) {
		return nil
	}

	return &handler.versions[version-1]
}

func hash(handler *handlerVersion, opts interface{}) ([]byte, error) {
	hashStruct := struct {
		Path    string      `json:"p"`
		Version string      `json:"v"`
		Query   interface{} `json:"q,omitempty"`
	}{
		Path:    handler.path,
		Version: handler.Version,
		Query:   opts,
	}

	return json.Marshal(&hashStruct)
}

func (hm *HandlersManager) lookupCache(ctx context.Context, t reflect.Type, key string) (interface{}, bool) {
	if v, hit := hm.cache.Get(key); hit {
		if t.Kind() == reflect.Ptr {
			val := reflect.ValueOf(v)
			ptr := reflect.New(val.Type())
			ptr.Elem().Set(val)
			v = ptr.Interface()
		}
		return v, true
	} else {
		return nil, false
	}
}

func (hm *HandlersManager) cacheResponse(key string, response interface{}) {
	v := response
	val := reflect.ValueOf(v)
	if val.Kind() == reflect.Ptr {
		v = val.Elem().Interface()
	}
	hm.cache.Put(key, v, hm.cacheTTL)
}

func (hm *HandlersManager) CallHandler(ctx context.Context, handler *handlerVersion, parameters IHandlerParameters) (interface{}, *CallHandlerError) {
	methodFunc := handler.method

	optsType := methodFunc.Type.In(2).Elem()
	params, err := prepareParameters(parameters, handler.Parameters, optsType)
	if err != nil {
		return nil, &CallHandlerError{ErrorInParameters, err}
	}

	in := []reflect.Value{reflect.ValueOf(handler.handlerStruct), reflect.ValueOf(ctx), params}

	paramIf := params.Interface()

	useCache := handler.UseCache && hm.cacheTTL != 0
	var cacheKey string
	if useCache {
		if h, err := hash(handler, paramIf); err == nil {
			cacheKey = string(h)
			responseType := methodFunc.Type.Out(0)
			if val, hit := hm.lookupCache(ctx, responseType, cacheKey); hit {
				return val, nil
			}
		}
	}

	out := methodFunc.Func.Call(in)

	if out[1].IsNil() {
		val := out[0].Interface()
		if useCache {
			hm.cacheResponse(cacheKey, val)
		}
		return val, nil
	}

	err = out[1].Interface().(error)

	switch internalErr := err.(type) {
	case *HandlerError:
		return nil, &CallHandlerError{ErrorReturnedFromCall, err}
	case *CallHandlerError:
		return nil, internalErr
	default:
		return nil, &CallHandlerError{ErrorUnknown, err}
	}
}

func (hm *HandlersManager) GetHandlersPaths() []string {
	res := make([]string, 0, len(hm.handlers))

	for p := range hm.handlers {
		res = append(res, p)
	}

	return res
}

func (hm *HandlersManager) GetHandlerInfo(path string) *handlerInfo {
	handler := hm.getHandlerByPath(path)
	if handler == nil {
		return nil
	}

	return &handlerInfo{
		Handler:     handler,
		Path:        path,
		Caption:     handler.handlerStruct.Caption(),
		Description: handler.handlerStruct.Description(),
		Versions:    handler.versions,
	}
}

func (hm *HandlersManager) getHandlerByPath(path string) *handlerEntity {
	return hm.handlers[path]
}

func prepareParameters(handlerParameters IHandlerParameters, parameters []handlerParameter, parametersStructType reflect.Type) (reflect.Value, error) {
	resPtr := reflect.New(parametersStructType)
	res := resPtr.Elem()

	existingHandlerMethods := reflect.TypeOf(handlerParameters)

	for _, param := range parameters {
		if !handlerParameters.IsExists(param.GetKey()) {
			if param.IsRequired {
				return reflect.ValueOf(nil), fmt.Errorf("Missed required field '%s'", param.GetKey())
			}
			continue
		}

		method := existingHandlerMethods.Method(param.getMethod.Index)
		retValues := method.Func.Call([]reflect.Value{reflect.ValueOf(handlerParameters), reflect.ValueOf(param.GetKey())})
		if len(retValues) > 1 && !retValues[1].IsNil() {
			return reflect.ValueOf(nil), retValues[1].Interface().(error)
		}

		structField := res.FieldByIndex(param.structField.Index)
		if structField.Kind() == reflect.Ptr {
			structField.Set(reflect.New(structField.Type().Elem()))
			structField = structField.Elem()
		}
		structField.Set(retValues[0])
	}

	return resPtr, nil
}

func findParameterGetMethod(handlerMethodsType reflect.Type, field reflect.Type) (reflect.Method, bool) {
	var name []rune
	switch field.Kind() {
	case reflect.Array, reflect.Slice:
		name = []rune(field.Elem().Kind().String() + "Slice")
	case reflect.Ptr:
		name = []rune(field.Elem().Kind().String())
	default:
		name = []rune(field.Kind().String())
	}
	name[0] = unicode.ToUpper(name[0])
	methodName := "Get" + string(name)

	return handlerMethodsType.MethodByName(methodName)
}
