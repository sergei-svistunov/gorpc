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
}

var handlerCallFuncTemplate = []byte(`
func (api *>>>API_NAME<<<) >>>HANDLER_NAME<<<(ctx context.Context>>>INPUT_PARAMS<<<) (>>>RETURNED_TYPE<<<, error) {
    var result >>>RETURNED_TYPE<<<
    params := map[string]interface{}{>>>MAPPED_INPUT_PARAMS<<<}

    err := api.get(ctx, ">>>HANDLER_PATH<<<", params, &result)
	return result, err
}

`)
var staticLogicTemplate = `
type >>>API_NAME<<< struct {
	client        *http.Client
	serviceCaller *ExternalServiceCaller
	ServiceName string
}

type httpSessionResponse struct {
	Result string      ` + "`" + `json:"result"` + "`" + `
	Data   json.RawMessage ` + "`" + `json:"data"` + "`" + `
	Error  string      ` + "`" + `json:"error"` + "`" + `
}

type IBalancer interface{
    Next() (string, error)
}

func New>>>API_NAME<<<(balancer IBalancer, apiTimeout int) *>>>API_NAME<<< {
	serviceName := ">>>SERVICE_NAME<<<"

	serviceCaller := &ExternalServiceCaller{
		Name:           strings.Title(serviceName),
		Balancer:       balancer,
	}
	return &>>>API_NAME<<<{
		client: &http.Client{
			Transport: &http.Transport{
				//DisableCompression: true,
				MaxIdleConnsPerHost: 20,
			},
			Timeout: time.Duration(apiTimeout) * time.Second,
		},
		serviceCaller: serviceCaller,
		ServiceName:   serviceName,
	}
}

func (api *>>>API_NAME<<<) get(ctx context.Context, path string, params map[string]interface{}, buf interface{}) error {
	values := ToURLValues(params)
	return api.getWithValues(ctx, path, values, buf)
}

func (api *>>>API_NAME<<<) getWithValues(ctx context.Context, path string, values url.Values, buf interface{}) error {
    sessionRequest := &SessionRequest{Path: path, Params: values}
	return api.serviceCaller.Call(ctx, sessionRequest, func(serviceURL string) (interface{}, error) {
		r, err := http.NewRequest("GET", CreateRawURL(serviceURL, path, values), nil)
		if err != nil {
			return nil, err
		}

		wrapper := httpSessionResponse{}
		if err := Do(api.client, r, &wrapper); err != nil {
			return nil, err
		}
        if err := json.Unmarshal(wrapper.Data, buf); err != nil {
            return nil, err
        }

		return buf, nil
	})
}

// ToURLValues converts map to url query.
func ToURLValues(params map[string]interface{}) url.Values {
	var values url.Values
	if len(params) > 0 {
		values = url.Values{}
		for k, v := range params {
			values.Set(k, fmt.Sprintf("%v", v))
		}
	}
	return values
}

func CreateRawURL(url, path string, values url.Values) string {
	rawURL := strings.TrimRight(url, "/") + "/" + strings.TrimLeft(path, "/")
	if len(values) > 0 {
		rawURL += "?" + values.Encode()
	}
	return rawURL
}

func Do(client *http.Client, request *http.Request, buf interface{}) (err error) {
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
	handlerCallFuncTemplate = regexp.MustCompilePOSIX(">>>API_NAME<<<").ReplaceAll(handlerCallFuncTemplate, []byte(strings.Title(serviceName)))
	return nil
}

func generateAdapterMethods(structsBuf *bytes.Buffer) []byte {
	result := &bytes.Buffer{}

	for path, handlerInfo := range path2HandlerInfoMapping {
		var method []byte
		method = regexp.MustCompilePOSIX(">>>HANDLER_PATH<<<").ReplaceAll(handlerCallFuncTemplate, []byte(path))
		path = strings.Replace(strings.Title(path), "/", "", -1) //TODO convert to CamelCaseName
		method = regexp.MustCompilePOSIX(">>>HANDLER_NAME<<<").ReplaceAll(method, []byte(path))
		method = regexp.MustCompilePOSIX(">>>RETURNED_TYPE<<<").ReplaceAll(method, []byte(handlerInfo.Output))
		method = regexp.MustCompilePOSIX(">>>INPUT_PARAMS<<<").ReplaceAll(method, generateInputParamsRow(handlerInfo.Params, structsBuf))
		method = regexp.MustCompilePOSIX(">>>MAPPED_INPUT_PARAMS<<<").ReplaceAll(method, generateMappedInputParamsString(handlerInfo.Params))

		result.Write(method)
	}

	return result.Bytes()
}

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
