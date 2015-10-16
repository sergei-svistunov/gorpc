package gorpc

import (
	"errors"
	"fmt"
	"reflect"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"unicode"

	"golang.org/x/net/context"
)

type handlerEntity struct {
	path          string
	versions      []handlerVersion
	handlerStruct IHandler
}

type HandlersManagerCallbacks struct {
	// OnHandlerRegistration will be called only one time for each handler version while handler registration is in progress
	OnHandlerRegistration func(path string, method reflect.Method) (extraData interface{})

	// OnError will be called if any error occures while CallHandler() method is in processing
	OnError func(ctx context.Context, err error)

	// OnSuccess will be called if CallHandler() method is successfully finished
	OnSuccess func(ctx context.Context, result interface{})

	// AppendInParams will be called for each handler call and can append extra parameters to params
	AppendInParams func(ctx context.Context, preparedParams []reflect.Value, extraData interface{}) (context.Context, []reflect.Value, error)
}

type HandlersManager struct {
	handlers        map[string]*handlerEntity
	handlerVersions map[string]*handlerVersion
	handlersPath    string
	callbacks       HandlersManagerCallbacks
}

func NewHandlersManager(handlersPath string, callbacks HandlersManagerCallbacks) *HandlersManager {
	return &HandlersManager{
		handlers:        make(map[string]*handlerEntity),
		handlerVersions: make(map[string]*handlerVersion),
		handlersPath:    strings.TrimSuffix(handlersPath, "/"),
		callbacks:       callbacks,
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

		handlerMethodPrefix := "V" + strconv.Itoa(v)
		vMethodType, _ := handlerType.MethodByName(handlerMethodPrefix)
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
		version.path = handlerPath
		version.method = vMethodType
		version.handlerStruct = h

		route := fmt.Sprintf("%s/%s/", handlerPath, version.Version)

		if callback := hm.callbacks.OnHandlerRegistration; callback != nil {
			version.ExtraData = callback(route, vMethodType)
		}

		hm.handlerVersions[route] = version

		_, version.UseCache = handlerType.MethodByName(handlerMethodPrefix + "UseCache")
		_, flatRequest := handlerType.MethodByName(handlerMethodPrefix + "ConsumeFlatRequest")

		if vMethodType.Type.NumOut() != 2 {
			return &CallHandlerError{ErrorInParameters, fmt.Errorf("Invalid count of output parameters for version number %d of handler %s", handlerVersion, handlerPath)}
		}

		if vMethodType.Type.Out(1).String() != "error" {
			return &CallHandlerError{ErrorInParameters, fmt.Errorf("Second output parameter should be error (handler %s version number %d)", handlerPath, handlerVersion)}
		}

		version.Request = handlerRequest{
			Type: paramsType,
			Flat: flatRequest,
		}
		if version.Request.Type.Kind() == reflect.Ptr {
			version.Request.Type = version.Request.Type.Elem()
		}

		// TODO: check response object for unexported fields here. Move that code out of docs.go
		version.Response = vMethodType.Type.Out(0)

		err := processParametersInfo(&version.Request, existingHandlerMethods)
		if err != nil {
			return fmt.Errorf("%s (handler %s, version number %d)", err.Error(), handlerPath, handlerVersion)
		}

		// check and prepare errors types for handler
		errMethod, found := handlerType.MethodByName(handlerMethodPrefix + "ErrorsVar")
		if found {
			errMethodType := errMethod.Type
			if errMethodType.NumOut() == 0 {
				return fmt.Errorf("V%dErrorsVar() method of handler %s should return errors types struct", handlerVersion, handlerPath)
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

func processParametersInfo(request *handlerRequest, existingHandlerMethods reflect.Type) error {
	params, err := processParamFields(request.Type, existingHandlerMethods, nil, request.Flat)
	if err != nil {
		return err
	}
	request.Fields = params
	return nil
}

func processParamFields(fieldType, existingHandlerMethods reflect.Type, path []string, flat bool) ([]handlerParameter, error) {
	var parameters []handlerParameter
	if fieldType.Kind() == reflect.Ptr {
		fieldType = fieldType.Elem()
	}
	for i := 0; i < fieldType.NumField(); i++ {
		fieldType := fieldType.Field(i)

		parameter := handlerParameter{
			Key:         fieldType.Tag.Get("key"),
			Path:        path,
			Name:        fieldType.Name,
			RawType:     fieldType.Type,
			structField: fieldType,
			Description: fieldType.Tag.Get("description"),
			IsRequired:  fieldType.Type.Kind() != reflect.Ptr,
		}

		if parameter.Key == "" || parameter.Key == "-" {
			return nil, fmt.Errorf("tag \"key\" must be specified for parameter %q", fieldType.Name)
		}

		if unicode.IsLower(rune(fieldType.Name[0])) {
			return nil, fmt.Errorf("Parameters field %s is private", parameter.Name)
		}

		if parameter.Description == "" {
			return nil, fmt.Errorf("Opt %s does not have description", fieldType.Name)
		}

		t := fieldType.Type
		if t.Kind() == reflect.Ptr {
			t = t.Elem()
		}
		if t.Kind() != reflect.Struct {
			paramGetMethod, exist := findParameterGetMethod(existingHandlerMethods, fieldType.Type)
			if !exist {
				return nil, fmt.Errorf("Type %s does not supported", fieldType.Type.Kind().String())
			}
			parameter.getMethod = paramGetMethod

		} else {
			if flat {
				return nil, fmt.Errorf("Deep nesting is not supported, type %s marked flat", fieldType.Name)
			}

			var path []string
			if len(parameter.Path) > 0 {
				path = append(path, parameter.Path...)
			}
			path = append(path, parameter.Key)
			var err error
			parameter.Fields, err = processParamFields(t, existingHandlerMethods, path, false)
			if err != nil {
				return nil, err
			}
		}

		parameters = append(parameters, parameter)
	}
	return parameters, nil
}

// FindHandler returns a handler by given non-versioned path and given version
// number
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

// FindHandlerByRoute returns a handler by fully qualified route to that
// particular version of the handler
func (hm *HandlersManager) FindHandlerByRoute(route string) *handlerVersion {
	if !strings.HasSuffix(route, "/") {
		route += "/"
	}
	return hm.handlerVersions[route]
}

func (hm *HandlersManager) PrepareParameters(ctx context.Context, handler *handlerVersion, parameters IHandlerParameters) (reflect.Value, error) {
	if err := parameters.Parse(); err != nil {
		return reflect.ValueOf(nil), &CallHandlerError{ErrorInParameters, err}
	}
	resPtr := reflect.New(handler.Request.Type)
	res := resPtr.Elem()
	err := prepareParameters(res, parameters, handler.Request.Fields, handler.Request.Type)
	if err != nil {
		return reflect.ValueOf(nil), &CallHandlerError{ErrorInParameters, err}
	}
	return resPtr, nil
}

func (hm *HandlersManager) CallHandler(ctx context.Context, handler *handlerVersion, params reflect.Value) (interface{}, *CallHandlerError) {
	in := []reflect.Value{reflect.ValueOf(handler.handlerStruct), reflect.ValueOf(ctx), params}

	if callback := hm.callbacks.AppendInParams; callback != nil {
		var err error
		ctx, in, err = callback(ctx, in, handler.ExtraData)
		if err != nil {
			return nil, &CallHandlerError{
				Type: ErrorReturnedFromCall,
				Err:  err,
			}
		}
	}

	out := handler.method.Func.Call(in)

	if out[1].IsNil() {
		val := out[0].Interface()
		if callback := hm.callbacks.OnSuccess; callback != nil {
			callback(ctx, val)
		}
		return val, nil
	}

	err := out[1].Interface().(error)
	if callback := hm.callbacks.OnError; callback != nil {
		callback(ctx, err)
	}

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

func prepareParameters(res reflect.Value, handlerParameters IHandlerParameters, parameters []handlerParameter, parametersStructType reflect.Type) error {
	existingHandlerMethods := reflect.TypeOf(handlerParameters)

	for _, param := range parameters {
		if !handlerParameters.IsExists(param.Path, param.GetKey()) {
			if param.IsRequired {
				return fmt.Errorf("Missed required field '%s'", param.GetKey())
			}
			continue
		}

		structField := res.FieldByIndex(param.structField.Index)
		if structField.Kind() == reflect.Ptr {
			structField.Set(reflect.New(structField.Type().Elem()))
			structField = structField.Elem()
		}

		t := param.RawType
		if t.Kind() == reflect.Ptr {
			t = t.Elem()
		}
		if t.Kind() != reflect.Struct {
			method := existingHandlerMethods.Method(param.getMethod.Index)
			retValues := method.Func.Call([]reflect.Value{reflect.ValueOf(handlerParameters), reflect.ValueOf(param.Path), reflect.ValueOf(param.GetKey())})
			if len(retValues) > 1 && !retValues[1].IsNil() {
				return retValues[1].Interface().(error)
			}
			structField.Set(retValues[0])
		} else {
			err := prepareParameters(structField, handlerParameters, param.Fields, param.RawType)
			if err != nil {
				return err
			}
		}
		// TODO: map, array, whatever
	}

	return nil
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
