package adapter

func init() {
	RegisterComponent(
		NewComponent(
			">>>STATIC_LOGIC<<<",
			mainImports,
			staticLogicTemplate,
			prepareStaticLogic),
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
	Result string      `+"`"+`json:"result"`+"`"+`
	Data   json.RawMessage `+"`"+`json:"data"`+"`"+`
	Error  string      `+"`"+`json:"error"`+"`"+`
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

func prepareStaticLogic(codePtr *string) error {
	// TODO replace service name in template
	return nil
}
