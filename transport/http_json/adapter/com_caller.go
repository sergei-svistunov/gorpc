package adapter

func init() {
	RegisterComponent(
		NewComponent(">>>CALLER<<<", callerImports, callerCode, nil),
	)
}

var callerImports = []string{
	"fmt",
	"runtime",
	"golang.org/x/net/context",
}

var callerCode = `
// SessionRequest contains session information for logging.
type SessionRequest struct {
	URL    string
	Path   string
	Params interface{}
}

type ExternalServiceCaller struct {
	Name     string
	Balancer IBalancer
}

func (service *ExternalServiceCaller) Call(ctx context.Context, sessionRequest *SessionRequest, caller func(string) (interface{}, error)) (err error) {
	if sessionRequest.URL == "" {
		var serviceURL string
		serviceURL, err = service.Balancer.Next()
		if err != nil {
			err = fmt.Errorf("could not locate service: %v", err)
			return
		}
		sessionRequest.URL = serviceURL
	}

	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("panic while calling %q service: %v", service.Name, r)
		}
	}()

	//var response interface{}
	if _, err = caller(sessionRequest.URL); err != nil {
		return
	}

	return
}

`
