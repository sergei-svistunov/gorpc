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
	`commonapi "lazada_api/common/api"`,
	"golang.org/x/net/context",
	"net/http",
	"fmt",
	"net/url",
	"strings",
	"time",
	"encoding/json",
}

var handlerCallFuncTemplate = `
func (api *ExternalAPI) Call>>>HANDLER_NAME<<<(ctx context.Context>>>INPUT_PARAMS<<<) (>>>RETURNED_TYPE<<<, error) {
    var result >>>RETURNED_TYPE<<<
    params := map[string]interface{}{>>>MAPPED_INPUT_PARAMS<<<}

    err := api.get(ctx, ">>>HANDLER_PATH<<<", params, &result)
	return result, err
}

`
var staticLogicTemplate = `
type ExternalAPI struct {
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

func NewExternalAPI(balancer IBalancer, venture string, environment string, apiTimeout int) *ExternalAPI {
	serviceName := ">>>SERVICE_NAME<<<"

	serviceCaller := &ExternalServiceCaller{
		Name:           strings.Title(serviceName),
		Balancer:       balancer,
	}
	return &ExternalAPI{
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

func (api *ExternalAPI) get(ctx context.Context, path string, params map[string]interface{}, buf interface{}) error {
	values := ToURLValues(params)
	return api.getWithValues(ctx, path, values, buf)
}

func (api *ExternalAPI) getWithValues(ctx context.Context, path string, values url.Values, buf interface{}) error {
    sessionRequest := &SessionRequest{Path: path, Params: values}
	return api.serviceCaller.Call(ctx, sessionRequest, func(serviceURL string) (interface{}, error) {
		r, err := http.NewRequest("GET", CreateRawURL(serviceURL, path, values), nil)
		if err != nil {
			return nil, err
		}

		wrapper := httpSessionResponse{}
		if err := commonapi.Do(api.client, r, &wrapper); err != nil {
			if apiErr, ok := err.(commonapi.ErrorResponse); ok {
				err = ServiceError{
					Code:    int(apiErr.ErrorCode),
					Message: apiErr.ErrorMessage,
				}
			}
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

`

func replaceServiceName(codePtr *[]byte) error {
	if codePtr == nil {
		return fmt.Errorf("Code pointer is nil")
	}

	*codePtr = regexp.MustCompilePOSIX(">>>SERVICE_NAME<<<").ReplaceAll(*codePtr, []byte(serviceName))
	return nil
}

func generateAdapterMethods(structsBuf *bytes.Buffer) []byte {
	result := &bytes.Buffer{}

	for path, handlerInfo := range path2HandlerInfoMapping {
		var method []byte
		method = regexp.MustCompilePOSIX(">>>HANDLER_PATH<<<").ReplaceAll([]byte(handlerCallFuncTemplate), []byte(path))
		path = strings.Replace(path, "/", "_", -1) //TODO convert to CamelCaseName
		method = regexp.MustCompilePOSIX(">>>HANDLER_NAME<<<").ReplaceAll(method, []byte(strings.Title(path)))
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
