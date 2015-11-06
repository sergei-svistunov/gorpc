package adapter

import (
	"bytes"
	"fmt"
	"github.com/sergei-svistunov/gorpc"
	"regexp"
	"strings"
)

func init() {
	RegisterComponent(
		NewComponent(
			">>>STATIC_LOGIC<<<",
			mainImports,
			staticLogicTemplate,
			replaceServiceName),
	)
}

var mainImports = []string{
	"golang.org/x/net/context",
	"net/http",
	"fmt",
	"net/url",
	"strings",
	"time",
	"encoding/json",
	"bytes",
	"io/ioutil",
}

var handlerCallGetFuncTemplate = []byte(`
func (api *>>>API_NAME<<<) >>>HANDLER_NAME<<<(ctx context.Context>>>INPUT_PARAMS<<<) (>>>RETURNED_TYPE<<<, error) {
    var result >>>RETURNED_TYPE<<<
    params := map[string]interface{}{>>>MAPPED_INPUT_PARAMS<<<}

    err := api.get(ctx, ">>>HANDLER_PATH<<<", params, &result)
	return result, err
}

`)

var staticLogicTemplate = `
type httpSessionResponse struct {
	Result string      ` + "`" + `json:"result"` + "`" + `
	Data   json.RawMessage ` + "`" + `json:"data"` + "`" + `
	Error  string      ` + "`" + `json:"error"` + "`" + `
}

type sessionRequest struct {
	URL    string
	Path   string
	Params interface{}
}

type IBalancer interface{
    Next() (string, error)
}

type >>>API_NAME<<< struct {
	client        *http.Client
	ServiceName string
	balancer IBalancer
}

func New>>>API_NAME<<<(balancer IBalancer, apiTimeout int, serviceName string) *>>>API_NAME<<< {
	return &>>>API_NAME<<<{
		client: &http.Client{
			Transport: &http.Transport{
				//DisableCompression: true,
				MaxIdleConnsPerHost: 20,
			},
			Timeout: time.Duration(apiTimeout) * time.Second,
		},
		ServiceName:   serviceName,
		balancer: balancer,
	}
}

func (api *>>>API_NAME<<<) get(ctx context.Context, path string, params map[string]interface{}, buf interface{}) error {
	values := toURLValues(params)
	return api.getWithValues(ctx, path, values, buf)
}

func (api *>>>API_NAME<<<) getWithValues(ctx context.Context, path string, values url.Values, buf interface{}) error {
    apiURL, err := api.balancer.Next()
    if err != nil {
        return err
    }
    sessionRequest := &sessionRequest{
        Path: path,
        Params: values,
        URL: apiURL,
    }
	return api.call(ctx, sessionRequest, func(serviceURL string) (interface{}, error) {
		r, err := http.NewRequest("GET", createRawURL(serviceURL, path, values), nil)
		if err != nil {
			return nil, err
		}

		wrapper := httpSessionResponse{}
		if err := do(api.client, r, &wrapper); err != nil {
			return nil, err
		}
        if err := json.Unmarshal(wrapper.Data, buf); err != nil {
            return nil, err
        }

		return buf, nil
	})
}

func (api *>>>API_NAME<<<) set(ctx context.Context, path string, data interface{}, buf interface{}) error {
    apiURL, err := api.balancer.Next()
    if err != nil {
        return err
    }
	sessionRequest := &sessionRequest{
	    Path: path,
	    Params: data,
	    URL: apiURL,
    }
	return api.call(ctx, sessionRequest, func(serviceURL string) (interface{}, error) {
		b := bytes.NewBuffer(nil)
		encoder := json.NewEncoder(b)
		if err := encoder.Encode(data); err != nil {
			return nil, fmt.Errorf("could not marshal data %+v: %v", data, err)
		}

		r, err := http.NewRequest("POST", createRawURL(serviceURL, path, nil), b)
		if err != nil {
			return nil, err
		}
		if err := do(api.client, r, buf); err != nil {
			return nil, err
		}
		return buf, nil
	})
}

func (api *>>>API_NAME<<<) call(ctx context.Context, sessionRequest *sessionRequest, caller func(string) (interface{}, error)) (err error) {
    if sessionRequest.URL == "" {
        return fmt.Errorf("Service URL is not defined")
    }

    defer func() {
        if r := recover(); r != nil {
            err = fmt.Errorf("panic while calling %q service: %v", api.ServiceName, r)
        }
    }()

    if _, err = caller(sessionRequest.URL); err != nil {
        return
    }

    return
}

func toURLValues(params map[string]interface{}) url.Values {
	var values url.Values
	if len(params) > 0 {
		values = url.Values{}
		for k, v := range params {
			values.Set(k, fmt.Sprintf("%v", v))
		}
	}
	return values
}

func createRawURL(url, path string, values url.Values) string {
	rawURL := strings.TrimRight(url, "/") + "/" + strings.TrimLeft(path, "/")
	if len(values) > 0 {
		rawURL += "?" + values.Encode()
	}
	return rawURL
}

func do(client *http.Client, request *http.Request, buf interface{}) (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("panic in request %q: %v", request.RequestURI, r)
		}
	}()

	// Run
	var response *http.Response
	if response, err = client.Do(request); err != nil {
		return err
	}
	defer response.Body.Close()

	// Handle error
	if response.StatusCode != http.StatusOK {
	    switch response.StatusCode {
	    // TODO separate error types for different status codes (and different callbacks)
	    /*
        case http.StatusForbidden:
        case http.StatusBadGateway:
        case http.StatusBadRequest:
        */
        default:
            return fmt.Errorf("Request %q failed. Server returns status code %d", request.RequestURI, response.StatusCode)
        }
	}

	// Read response
	var result []byte
	if result, err = ioutil.ReadAll(response.Body); err != nil {
		return err
	}

	if err = json.Unmarshal(result, buf); err != nil {
		return fmt.Errorf("request %q failed to decode response %q: %v", request.RequestURI, string(result), err)
	}

	return nil
}


`

func replaceServiceName(codePtr *[]byte) error {
	if codePtr == nil {
		return fmt.Errorf("Code pointer is nil")
	}

	*codePtr = regexp.MustCompilePOSIX(">>>SERVICE_NAME<<<").ReplaceAll(*codePtr, []byte(strings.ToLower(serviceName)))
	*codePtr = regexp.MustCompilePOSIX(">>>API_NAME<<<").ReplaceAll(*codePtr, []byte(strings.Title(serviceName)))
	handlerCallGetFuncTemplate = regexp.MustCompilePOSIX(">>>API_NAME<<<").ReplaceAll(handlerCallGetFuncTemplate, []byte(strings.Title(serviceName)))
	handlerCallPostFuncTemplate = regexp.MustCompilePOSIX(">>>API_NAME<<<").ReplaceAll(handlerCallPostFuncTemplate, []byte(strings.Title(serviceName)))
	return nil
}

func generateAdapterMethods(structsBuf *bytes.Buffer) []byte {
	result := &bytes.Buffer{}

	for path, handlerInfo := range path2HandlerInfoMapping {
		var method []byte
		/*
			method = regexp.MustCompilePOSIX(">>>HANDLER_PATH<<<").ReplaceAll(handlerCallGetFuncTemplate, []byte(path))
			path = strings.Replace(strings.Title(path), "/", "", -1) //TODO convert to CamelCaseName
			method = regexp.MustCompilePOSIX(">>>HANDLER_NAME<<<").ReplaceAll(method, []byte(path))
			method = regexp.MustCompilePOSIX(">>>RETURNED_TYPE<<<").ReplaceAll(method, []byte(handlerInfo.Output))
			method = regexp.MustCompilePOSIX(">>>INPUT_PARAMS<<<").ReplaceAll(method, generateInputParamsRow(handlerInfo.Params, structsBuf))
			method = regexp.MustCompilePOSIX(">>>MAPPED_INPUT_PARAMS<<<").ReplaceAll(method, generateMappedInputParamsString(handlerInfo.Params))
		*/
		method = regexp.MustCompilePOSIX(">>>HANDLER_PATH<<<").ReplaceAll(handlerCallPostFuncTemplate, []byte(path))
		path = strings.Replace(strings.Title(path), "/", "", -1) //TODO convert to CamelCaseName
		method = regexp.MustCompilePOSIX(">>>HANDLER_NAME<<<").ReplaceAll(method, []byte(path))
		method = regexp.MustCompilePOSIX(">>>INPUT_TYPE<<<").ReplaceAll(method, []byte(handlerInfo.Input))
		method = regexp.MustCompilePOSIX(">>>RETURNED_TYPE<<<").ReplaceAll(method, []byte(handlerInfo.Output))

		result.Write(method)
	}

	return result.Bytes()
}

var handlerCallPostFuncTemplate = []byte(`
func (api *>>>API_NAME<<<) >>>HANDLER_NAME<<<(ctx context.Context, data >>>INPUT_TYPE<<<) (>>>RETURNED_TYPE<<<, error) {
    var result >>>RETURNED_TYPE<<<

    err := api.set(ctx, ">>>HANDLER_PATH<<<", data, &result)
	return result, err
}

`)

func generateInputParamsRow(params []gorpc.HandlerParameter, additionalStructsBuf *bytes.Buffer) []byte {
	var s string
	for _, param := range params {
		if param.RawType == nil {
			// TODO log error?
			//log.Errorf("Param %#v is nil", param)
			continue
		}
		t, extraStructs := detectTypeName(param.RawType, nil)
		s += (", " + param.Name + " " + t)
		if len(extraStructs) > 0 {
			for i := range extraStructs {
				_, err := convertStructToCode(extraStructs[i], additionalStructsBuf, nil)
				if err != nil {
					// TODO log error?
					panic(err)
				}
			}
		}
	}
	return []byte(s)
}

func generateMappedInputParamsString(params []gorpc.HandlerParameter) []byte {
	var s string
	for _, param := range params {
		s += ("\"" + param.GetKey() + "\": " + param.Name + ",\n")
	}
	return []byte(s)
}
