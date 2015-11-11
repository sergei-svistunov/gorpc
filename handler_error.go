package gorpc

const (
	ErrorInParameters = iota
	ErrorReturnedFromCall
	ErrorInvalidMethod
	ErrorWriteResponse
	ErrorUnknown
)

type HandlerError struct {
	UserMessage string
	Err         error
	Code        int
}

func (e *HandlerError) Error() string {
	return e.Err.Error()
}

type CallHandlerError struct {
	Type int
	Err  error
}

func (e *CallHandlerError) Error() string {
	return e.Err.Error()
}

func (e *CallHandlerError) UserMessage() string {
	if userErr, ok := e.Err.(*HandlerError); ok {
		if userErr.UserMessage != "" {
			return userErr.UserMessage
		}
	}

	return e.Err.Error()
}

func (e *CallHandlerError) ErrorCode() int {
	if userErr, ok := e.Err.(*HandlerError); ok {
		return userErr.Code
	}
	return 0
}
