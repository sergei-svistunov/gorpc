package adapter

var mainImports = []string{
	"bytes",
	"encoding/json",
	"fmt",
	"io/ioutil",
	"net/http",
	"net/url",
	"runtime",
	"strings",
	"time",
	"golang.org/x/net/context",
	"github.com/sergei-svistunov/gorpc/transport/cache",
	"github.com/mailru/easyjson",
}

var mainTemplate = []byte(`
// It's auto-generated file. It's not recommended to modify it.
package >>>PKG_NAME<<<

import (
    >>>IMPORTS<<<
)

type IBalancer interface {
    Next() (string, error)
}

type Callbacks struct {
	OnStart                func(ctx context.Context, req *http.Request) context.Context
	OnPrepareRequest       func(ctx context.Context, req *http.Request, data interface{}) context.Context
	OnResponseUnmarshaling func(ctx context.Context, req *http.Request, response *http.Response, result []byte)
	OnSuccess              func(ctx context.Context, req *http.Request, data interface{})
	OnError                func(ctx context.Context, req *http.Request, err error) error
	OnPanic                func(ctx context.Context, req *http.Request, r interface{}, trace []byte) error
	OnFinish               func(ctx context.Context, req *http.Request, startTime time.Time)
}

type >>>API_NAME<<< struct {
	client      *http.Client
	serviceName string
	balancer    IBalancer
	callbacks   Callbacks
	cache       cache.ICache
}

func (api *>>>API_NAME<<<) SetCache(c cache.ICache) *>>>API_NAME<<< {
	api.cache = c
	return api
}

func New>>>API_NAME<<<(client *http.Client, balancer IBalancer, callbacks Callbacks) *>>>API_NAME<<< {
	if client == nil {
		client = http.DefaultClient
	}
	return &>>>API_NAME<<<{
//		client: &http.Client{
//			Transport: &http.Transport{
//				//DisableCompression: true,
//				MaxIdleConnsPerHost: 20,
//			},
//			Timeout: apiTimeout,
//		},
		serviceName: ">>>API_NAME<<<",
		balancer:    balancer,
		callbacks:   callbacks,
		client:      client,
	}
}

>>>CLIENT_API<<<

// TODO: duplicates http_json.httpSessionResponse
// easyjson:json
type httpSessionResponse struct {
	Result string              ` + "`" + `json:"result"` + "`" + ` //OK or ERROR
	Data   easyjson.RawMessage ` + "`" + `json:"data"` + "`" + `
	Error  string              ` + "`" + `json:"error"` + "`" + `
}

func unmarshal(data []byte, r interface{}) error {
	if m, ok := r.(easyjson.Unmarshaler); ok {
		return easyjson.Unmarshal(data, m)
	}
	return json.Unmarshal(data, r)
}

func (api *>>>API_NAME<<<) set(ctx context.Context, path string, data interface{}, buf interface{}, handlerErrors map[string]int) (err error) {
	startTime := time.Now()

	var apiURL string
	var req *http.Request

	if api.callbacks.OnStart != nil {
		ctx = api.callbacks.OnStart(ctx, req)
	}

	defer func() {
		if api.callbacks.OnFinish != nil {
			api.callbacks.OnFinish(ctx, req, startTime)
		}

		if r := recover(); r != nil {
			const size = 64 << 10
			buf := make([]byte, size)
			n := runtime.Stack(buf, false)
			trace := buf[:n]

			err = fmt.Errorf("panic while calling %q service: %v", api.serviceName, r)
			if api.callbacks.OnPanic != nil {
				err = api.callbacks.OnPanic(ctx, req, r, trace)
			}
		}
	}()

	apiURL, err = api.balancer.Next()
	if err != nil {
		err = fmt.Errorf("could not locate service '%s': %v", api.serviceName, err)
		if api.callbacks.OnError != nil {
			err = api.callbacks.OnError(ctx, req, err)
		}
		return err
	}

	b := bytes.NewBuffer(nil)
	if m, ok := data.(easyjson.Marshaler); ok {
		_, err = easyjson.MarshalToWriter(m, b)
	} else {
		encoder := json.NewEncoder(b)
		err = encoder.Encode(data)
	}
	if err != nil {
		err = fmt.Errorf("could not marshal data %+v: %v", data, err)
		if api.callbacks.OnError != nil {
			err = api.callbacks.OnError(ctx, req, err)
		}
		return err
	}

	req, err = http.NewRequest("POST", createRawURL(apiURL, path, nil), b)
	if err != nil {
		if api.callbacks.OnError != nil {
			err = api.callbacks.OnError(ctx, req, err)
		}
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	if api.callbacks.OnPrepareRequest != nil {
		ctx = api.callbacks.OnPrepareRequest(ctx, req, data)
	}

	if err := api.doRequest(ctx, req, buf, handlerErrors); err != nil {
		if api.callbacks.OnError != nil {
			err = api.callbacks.OnError(ctx, req, err)
		}
		return err
	}

	if api.callbacks.OnSuccess != nil {
		api.callbacks.OnSuccess(ctx, req, buf)
	}

	return nil
}

func (api *>>>API_NAME<<<) setWithCache(ctx context.Context, path string, data interface{}, entry *cache.CacheEntry, handlerErrors map[string]int) error {
	if api.cache != nil && cache.IsTransportCacheEnabled(ctx) {
		cacheKey := getCacheKey(path, data)
		if cacheKey != nil {
			api.cache.Lock(cacheKey)
			defer api.cache.Unlock(cacheKey)
			cacheEntry := api.cache.Get(cacheKey)
			if cacheEntry != nil && cacheEntry.Body != nil {
				*entry = *cacheEntry
				return nil
			}
			if err := api.set(ctx, path, data, entry.Body, handlerErrors); err != nil {
				return err
			}
			ttl := cache.TTL(ctx)
			if p, ok := api.cache.(cache.TTLAwarePutter); ok && ttl > 0 {
				p.PutWithTTL(cacheKey, entry, ttl)
			} else {
				api.cache.Put(cacheKey, entry)
			}
			return nil
		}
	}
	return api.set(ctx, path, data, entry.Body, handlerErrors)
}

func createRawURL(url, path string, values url.Values) string {
	var buf bytes.Buffer
	buf.WriteString(strings.TrimRight(url, "/"))
	//buf.WriteRune('/')
	//buf.WriteString(strings.TrimLeft(path, "/"))
	// path must contain leading /
	buf.WriteString(path)
	if len(values) > 0 {
		buf.WriteRune('?')
		buf.WriteString(values.Encode())
	}
	return buf.String()
}

func (api *>>>API_NAME<<<) doRequest(ctx context.Context, request *http.Request, buf interface{}, handlerErrors map[string]int) error {
	return HTTPDo(ctx, api.client, request, func(response *http.Response, err error) error {
		// Run
		if err != nil {
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
				return fmt.Errorf("Request %q failed. Server returns status code %d", request.URL.RequestURI(), response.StatusCode)
			}
		}

		// Read response
		result, err := ioutil.ReadAll(response.Body)
		if err != nil {
			return err
		}

		if api.callbacks.OnResponseUnmarshaling != nil {
			api.callbacks.OnResponseUnmarshaling(ctx, request, response, result)
		}

		var mainResp httpSessionResponse
		if err := unmarshal(result, &mainResp); err != nil {
			return fmt.Errorf("request %q failed to decode response %q: %v", request.URL.RequestURI(), string(result), err)
		}
		if mainResp.Result == "OK" {
			if err := unmarshal(mainResp.Data, buf); err != nil {
				return fmt.Errorf("request %q failed to decode response data %+v: %v", request.URL.RequestURI(), mainResp.Data, err)
			}
			return nil
		}

		if mainResp.Result == "ERROR" {
			errCode, ok := handlerErrors[mainResp.Error]
			if ok {
				return &ServiceError{
					Code:    errCode,
					Message: mainResp.Error,
				}
			}
		}

		return fmt.Errorf("request %q returned incorrect response %q", request.URL.RequestURI(), string(result))
	})
}

// HTTPDo is taken and adapted from https://blog.golang.org/context
func HTTPDo(ctx context.Context, client *http.Client, req *http.Request, f func(*http.Response, error) error) error {
	c := make(chan error, 1)
	go func() { c <- f(client.Do(req)) }()
	select {
	case <-ctx.Done():
		if tr, ok := client.Transport.(canceler); ok {
			tr.CancelRequest(req)
			<-c // Wait for f to return.
		}
		return ctx.Err()
	case err := <-c:
		return err
	}
}

type canceler interface {
	CancelRequest(*http.Request)
}

// ServiceError uses to separate critical and non-critical errors which returns in external service response.
// For this type of error we shouldn't use 500 error counter for librato
type ServiceError struct {
	Code    int
	Message string
}

// Error method for implementing common error interface
func (err *ServiceError) Error() string {
	return err.Message
}

func getCacheKey(route string, params interface{}) []byte {
	buf := bytes.NewBufferString(route)
	var err error
	if m, ok := params.(easyjson.Marshaler); ok {
		_, err = easyjson.MarshalToWriter(m, buf)
	} else {
		encoder := json.NewEncoder(buf)
		err = encoder.Encode(params)
	}
	if err != nil {
		return nil
	}
	return buf.Bytes()
}
`)

var handlerCallPostFuncTemplate = []byte(`
func (api *>>>API_NAME<<<) >>>HANDLER_NAME<<<(ctx context.Context, options >>>INPUT_TYPE<<<) (>>>RETURNED_TYPE<<<, error) {
	var result >>>RETURNED_TYPE<<<
	var entry = cache.CacheEntry{Body: &result}
	err := api.setWithCache(ctx, ">>>HANDLER_PATH<<<", options, &entry, >>>HANDLER_ERRORS<<<)
	if result, ok := entry.Body.(*>>>RETURNED_TYPE<<<); ok {
		return *result, err
	}
	return result, err
}
`)
