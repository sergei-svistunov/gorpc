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

	"context"
)

var isSamePackagePathException = map[string]bool{"time": true}

type HandlerVersion *handlerVersion

type handlerEntity struct {
	path          string
	versions      []handlerVersion
	handlerStruct IHandler
}

type HandlersManagerCallbacks struct {
	// OnHandlerRegistration will be called only one time for each handler version while handler registration is in progress
	OnHandlerRegistration func(path string, method reflect.Method) (extraData interface{})

	// OnError will be called if any error occurs while CallHandler() method is in processing
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

func (hm *HandlersManager) Pkg() string {
	return hm.handlersPath
}

func (hm *HandlersManager) MustRegisterHandlers(handlers ...IHandler) {
	if err := hm.RegisterHandlers(handlers...); err != nil {
		panic(err)
	}
}

func (hm *HandlersManager) RegisterHandlers(handlers ...IHandler) error {
	for _, h := range handlers {
		if err := hm.RegisterHandler(h); err != nil {
			return err
		}
	}
	return nil
}

func (hm *HandlersManager) MustRegisterHandler(h IHandler) {
	if err := hm.RegisterHandler(h); err != nil {
		panic(err)
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

	typesUsageInHandlers := make(map[reflect.Type][]string)

	for i, v := range handlerVersionsIds {
		handlerVersion := i + 1
		if handlerVersion != v {
			return fmt.Errorf("You have missed version number %d of handler %s", handlerVersion, handlerPath)
		}

		handlerMethodPrefix := "V" + strconv.Itoa(v)
		handlerPathWithVersion := handlerPath + "/" + handlerMethodPrefix
		vMethodType, _ := handlerType.MethodByName(handlerMethodPrefix)
		numIn := vMethodType.Type.NumIn()
		if numIn != 3 && numIn != 4 {
			return fmt.Errorf("Invalid prototype for version number %d of handler %s", handlerVersion, handlerPath)
		}

		ctxType := vMethodType.Type.In(1)
		if ctxType.Kind() != reflect.Interface || !strings.HasSuffix(ctxType.PkgPath(), "context") || ctxType.Name() != "Context" {
			return fmt.Errorf("Invalid prototype for version number %d of handler %s. First argument must be \"Context\" from package \"context\"", handlerVersion, handlerPath)
		}

		paramsType := vMethodType.Type.In(2)
		if paramsType.Kind() != reflect.Ptr || paramsType.Elem().Kind() != reflect.Struct {
			return fmt.Errorf("Type of opts for version number %d of handler %s must be Ptr to Struct", handlerVersion, handlerPath)
		}

		err := validateStructure(handlerPtrType.PkgPath(), paramsType, handlerPathWithVersion+"(args)", typesUsageInHandlers)
		if err != nil {
			return fmt.Errorf("Handler '%s' version '%s' parameter: %s", handlerPath, vMethodType.Name, err)
		}

		version := &versions[i]
		version.Version = "v" + strconv.Itoa(v)
		version.path = handlerPath
		version.method = vMethodType
		version.handlerStruct = h

		version.Route = fmt.Sprintf("%s/%s/", handlerPath, version.Version)

		if callback := hm.callbacks.OnHandlerRegistration; callback != nil {
			version.ExtraData = callback(version.Route, vMethodType)
		}

		hm.handlerVersions[version.Route] = version

		_, deprecatedUseCache := handlerType.MethodByName(handlerMethodPrefix + "UseCache")
		_, deprecatedUseETag := handlerType.MethodByName(handlerMethodPrefix + "UseEtag")
		if deprecatedUseCache || deprecatedUseETag {
			panic("UseCache and UseEtag marker methods no longer supported. Use cache.EnableTrasportCache(), cache.DisableTrasportCache(), cache.EnableETag() and cache.DisableETag()")
		}

		if vMethodType.Type.NumOut() != 2 {
			return fmt.Errorf("Invalid count of output parameters for version number %d of handler %s", handlerVersion, handlerPath)
		}

		if vMethodType.Type.Out(1).String() != "error" {
			return fmt.Errorf("Second output parameter should be error (handler %s version number %d)", handlerPath, handlerVersion)
		}

		// TODO: check response object for unexported fields here. Move that code out of docs.go
		version.Response = vMethodType.Type.Out(0)

		err = validateStructure(handlerPtrType.PkgPath(), version.Response, handlerPathWithVersion+"(response)", typesUsageInHandlers)
		if err != nil {
			return fmt.Errorf("Handler '%s' version '%s' return value: %s", handlerPath, vMethodType.Name, err)
		}

		version.Request, err = processRequestType(paramsType)
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

	if err := checkCustomTypesInResponseResults(typesUsageInHandlers); err != nil {
		return err
	}

	hm.handlers[handlerPath] = &handlerEntity{
		path:          handlerPath,
		versions:      versions,
		handlerStruct: h,
	}

	return nil
}

func checkCustomTypesInResponseResults(typesUsageInHandlers map[reflect.Type][]string) error {
	if len(typesUsageInHandlers) == 0 {
		return nil
	}
	foundError := false
	errorText := "Following handlers use shared structures as args/response:"
	for t, handlers := range typesUsageInHandlers {
		if len(handlers) < 2 {
			continue
		}
		foundError = true
		for _, handler_path := range handlers {
			errorText += ("\n - " + handler_path + ": " + t.String())
		}
	}

	if foundError {
		return fmt.Errorf(errorText)
	}
	return nil
}

func validateStructure(packagePath string, basicType reflect.Type, handlerPathWithVersion string, typesUsageInHandlers map[reflect.Type][]string) error {
	if typesUsageInHandlers == nil {
		return fmt.Errorf("typesUsageInHandlers should not be nil")
	}
	if handlersList, exist := typesUsageInHandlers[basicType]; exist {
		for _, handlerPath := range handlersList {
			if handlerPath == handlerPathWithVersion {
				return nil
			}
		}
	}

	pkgPath := basicType.PkgPath()

	if basicType.Kind() == reflect.Slice || basicType.Kind() == reflect.Array {
		return validateStructure(packagePath, basicType.Elem(), handlerPathWithVersion, typesUsageInHandlers)
	} else if basicType.Kind() == reflect.Map {
		if err := validateStructure(packagePath, basicType.Key(), handlerPathWithVersion, typesUsageInHandlers); err != nil {
			return err
		}
		if err := validateStructure(packagePath, basicType.Elem(), handlerPathWithVersion, typesUsageInHandlers); err != nil {
			return err
		}
		return nil
	} else if basicType.Kind() == reflect.Ptr {
		return validateStructure(packagePath, basicType.Elem(), handlerPathWithVersion, typesUsageInHandlers)
	} else if len(pkgPath) == 0 || isPrimitiveType(basicType) {
		return nil
	} else if _, exception := isSamePackagePathException[pkgPath]; exception {
		return nil
	} else if pkgPath != packagePath {
		return fmt.Errorf(`Structure must be fully defined in the same package. Type '%s' is not.`, basicType)
	} else if basicType.Kind() == reflect.Struct {
		if _, exist := typesUsageInHandlers[basicType]; !exist {
			typesUsageInHandlers[basicType] = make([]string, 0)
		}
		typesUsageInHandlers[basicType] = append(typesUsageInHandlers[basicType], handlerPathWithVersion)
		for i := 0; i < basicType.NumField(); i++ {
			err := validateStructure(packagePath, basicType.Field(i).Type, handlerPathWithVersion, typesUsageInHandlers)
			if err != nil {
				return err
			}
		}
		return nil
	}

	return errors.New(`Unreachable code`)
}

func isPrimitiveType(t reflect.Type) bool {
	switch t.Kind() {
	case reflect.String, reflect.Bool, reflect.Int, reflect.Int8, reflect.Int16,
		reflect.Int32, reflect.Int64, reflect.Uint, reflect.Uint8, reflect.Uint16,
		reflect.Uint32, reflect.Uint64, reflect.Float32, reflect.Float64,
		reflect.Complex64, reflect.Complex128:
		return true
	default:
		return false
	}
}

func processRequestType(requestType reflect.Type) (*handlerRequest, error) {
	handlerParametersType := reflect.TypeOf(new(IHandlerParameters)).Elem()

	request := &handlerRequest{
		Type: requestType,
		// assume it's flat otherwise we set it to false in processParamFields
		Flat: true,
	}
	if request.Type.Kind() == reflect.Ptr {
		request.Type = request.Type.Elem()
	}

	var err error
	request.Fields, err = processParamFields(request, request.Type, handlerParametersType, nil, nil)
	if err != nil {
		return nil, err
	}

	return request, nil
}

func processParamFields(request *handlerRequest, fieldType, handlerParametersType reflect.Type, path []string, encountered map[reflect.Type][]HandlerParameter) ([]HandlerParameter, error) {
	if encountered == nil {
		encountered = make(map[reflect.Type][]HandlerParameter)
	}

	if fieldType.Kind() == reflect.Ptr {
		fieldType = fieldType.Elem()
	}

	if encountered[fieldType] != nil {
		return encountered[fieldType], nil
	}

	parameters := make([]HandlerParameter, fieldType.NumField())
	encountered[fieldType] = parameters

	for i := 0; i < fieldType.NumField(); i++ {
		fieldType := fieldType.Field(i)

		parameter := HandlerParameter{
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
		if t.Kind() != reflect.Struct && t.Kind() != reflect.Map && t.Kind() != reflect.Slice && t.Kind() != reflect.Array {
			paramGetMethod, exist := findParameterGetMethod(handlerParametersType, fieldType.Type)
			if !exist {
				return nil, fmt.Errorf("Type %s does not supported", fieldType.Type.Kind().String())
			}
			parameter.getMethod = paramGetMethod

		} else {
			var path []string
			if len(parameter.Path) > 0 {
				path = append(path, parameter.Path...)
			}
			path = append(path, parameter.Key)
			var err error
			if t.Kind() == reflect.Slice || t.Kind() == reflect.Array || t.Kind() == reflect.Map {
				t = t.Elem()
				path = nil
			}
			if t.Kind() == reflect.Ptr {
				t = t.Elem()
			}
			if t.Kind() == reflect.Struct {
				parameter.Fields, err = processParamFields(request, t, handlerParametersType, path, copyEncounteredMap(encountered))
				request.Flat = false
				if err != nil {
					return nil, err
				}
			}
		}

		parameters[i] = parameter
	}
	return parameters, nil
}

func copyEncounteredMap(m map[reflect.Type][]HandlerParameter) map[reflect.Type][]HandlerParameter {
	mCopy := make(map[reflect.Type][]HandlerParameter, len(m))
	for k, v := range m {
		mCopy[k] = v
	}
	return mCopy
}

// FindHandler returns a handler by given non-versioned path and given version
// number
func (hm *HandlersManager) FindHandler(path string, version int) HandlerVersion {
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
func (hm *HandlersManager) FindHandlerByRoute(route string) HandlerVersion {
	if !strings.HasSuffix(route, "/") {
		route += "/"
	}
	return hm.handlerVersions[route]
}

func (*HandlersManager) UnmarshalParameters(ctx context.Context, handler HandlerVersion,
	handlerParameters IHandlerParameters) (reflect.Value, error) {
	return unmarshalRequest(handler.Request, handlerParameters)
}

func unmarshalRequest(request *handlerRequest, handlerParameters IHandlerParameters) (reflect.Value, error) {
	if err := handlerParameters.Parse(); err != nil {
		return reflect.ValueOf(nil), &CallHandlerError{ErrorInParameters, err}
	}
	resPtr := reflect.New(request.Type)
	res := resPtr.Elem()
	err := unmarshalParameters(res, handlerParameters, request.Fields, request.Type)
	if err != nil {
		return reflect.ValueOf(nil), &CallHandlerError{ErrorInParameters, err}
	}
	return resPtr, nil
}

func (hm *HandlersManager) CallHandler(ctx context.Context, handler HandlerVersion, params reflect.Value) (interface{}, *CallHandlerError) {
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

		if handler.Response.Kind() == reflect.Slice && out[0].IsNil() {
			val = reflect.MakeSlice(handler.Response, 0, 0).Interface()
		}

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

func unmarshalParameters(res reflect.Value, handlerParameters IHandlerParameters, parameters []HandlerParameter,
	parametersStructType reflect.Type) error {

	handlerParametersType := reflect.TypeOf(handlerParameters)

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
		if t.Kind() == reflect.Struct {
			fields := make([]HandlerParameter, len(param.Fields))
			copy(fields, param.Fields)
			for i := range fields {
				fields[i].Path = append(param.Path, param.Key)
			}
			err := unmarshalParameters(structField, handlerParameters, fields, param.RawType)
			if err != nil {
				return err
			}

		} else if t.Kind() == reflect.Slice || t.Kind() == reflect.Array {
			container := structField
			ok, err := handlerParameters.TraverseSlice(param.Path, param.Key, func(i int, v interface{}) error {
				fields := param.Fields
				if t.Elem().Kind() == reflect.Struct {
					f, _ := processRequestType(t.Elem())
					fields = f.Fields
				}
				val, err := createContainerValue(t.Elem(), v,
					HandlerParameter{
						Path:    append(param.Path, param.Key),
						Key:     strconv.FormatInt(int64(i), 10),
						RawType: t.Elem(),
						Fields:  fields,
					},
					handlerParameters)
				if err != nil {
					return err
				}

				//ToDo: Fix it to right way, need change major version
				if container.Type().Elem().Kind() != val.Kind() && val.Kind() == reflect.String {
					newVal := reflect.New(container.Type().Elem()).Elem()
					switch container.Type().Elem().Kind() {
					case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
						if v, err := strconv.ParseInt(val.String(), 10, 64); err == nil {
							newVal.SetInt(v)
						} else {
							return err
						}
					case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
						if v, err := strconv.ParseUint(val.String(), 10, 64); err == nil {
							newVal.SetUint(v)
						} else {
							return err
						}
					default:
						return fmt.Errorf("Cannot convert to %s", container.Type().Elem().Kind().String())
					}
					val = newVal
				}

				container = reflect.Append(container, val)
				return nil
			})
			if err != nil {
				return err
			}
			if ok {
				structField.Set(container)
			}

		} else if t.Kind() == reflect.Map {
			container := reflect.MakeMap(t)
			ok, err := handlerParameters.TraverseMap(param.Path, param.Key, func(k string, v interface{}) error {
				val, err := createContainerValue(t.Elem(), v, param, handlerParameters)
				if err != nil {
					return err
				}
				container.SetMapIndex(reflect.ValueOf(k), val)
				return nil
			})
			if err != nil {
				return err
			}
			if ok {
				structField.Set(container)
			}

		} else {
			method := handlerParametersType.Method(param.getMethod.Index)
			retValues := method.Func.Call([]reflect.Value{reflect.ValueOf(handlerParameters), reflect.ValueOf(param.Path), reflect.ValueOf(param.GetKey())})
			if len(retValues) > 1 && !retValues[1].IsNil() {
				return retValues[1].Interface().(error)
			}
			structField.Set(retValues[0])
		}
	}
	return nil
}

func createContainerValue(t reflect.Type, v interface{}, param HandlerParameter, handlerParameters IHandlerParameters) (reflect.Value, error) {
	val := reflect.ValueOf(v)
	if t.Kind() == reflect.Ptr {
		t = val.Elem().Type()
	}
	if t.Kind() == reflect.Struct {
		val = reflect.New(t).Elem()
		err := unmarshalParameters(val, handlerParameters, param.Fields, t)
		if err != nil {
			return reflect.ValueOf(nil), err
		}
	}
	if t.Kind() == reflect.Slice {
		sliceVal := reflect.New(t).Elem()
		for i := 0; i < val.Len(); i++ {
			elemVal := reflect.New(t.Elem()).Elem()
			fields := param.Fields
			if t.Elem().Kind() == reflect.Struct {
				f, _ := processRequestType(t.Elem())
				fields = f.Fields
				for fi, _ := range fields {
					fields[fi].Path = append(param.Path, param.Key, strconv.FormatInt(int64(i), 10))
				}
			}
			err := unmarshalParameters(
				elemVal,
				handlerParameters,
				[]HandlerParameter{
					HandlerParameter{
						Path:    append(param.Path, param.Key),
						Key:     strconv.FormatInt(int64(i), 10),
						RawType: t.Elem(),
						Fields:  fields,
					},
				},
				t.Elem(),
			)
			if err != nil {
				return reflect.ValueOf(nil), err
			}

			sliceVal = reflect.Append(sliceVal, elemVal)

			//pretty.Println(val.Index(i).Interface())
		}
		val = sliceVal
	}
	return val, nil
}

func findParameterGetMethod(handlerParametersType reflect.Type, field reflect.Type) (reflect.Method, bool) {
	var name []rune
	switch field.Kind() {
	case reflect.Ptr:
		name = []rune(field.Elem().Kind().String())
	default:
		name = []rune(field.Kind().String())
	}
	name[0] = unicode.ToUpper(name[0])
	methodName := "Get" + string(name)
	return handlerParametersType.MethodByName(methodName)
}
