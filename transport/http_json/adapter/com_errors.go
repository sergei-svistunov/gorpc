package adapter

func init() {
	RegisterComponent(
		NewComponent(">>>ERRORS<<<", errorsImports, errorsCode, nil),
	)
}

var errorsImports = []string{
	"errors",
}

var errorsCode = `
// ErrInvalidResponseFormat is an error returned by unexpected external service's response.
var ErrInvalidResponseFormat = errors.New("Invalid response format from API")

// ServiceError uses to separate critical and non-critical errors which returns in external service response.
// For this type of error we shouldn't use 500 error counter for librato
type ServiceError struct {
	Code    int
	Message string
}

// Error method for implementing common error interface
func (err ServiceError) Error() string {
	return err.Message
}
`
