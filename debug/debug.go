package debug

import (
	"golang.org/x/net/context"
)

const DebugContextKey = "debug"

type Debug struct {
	Modules map[string]interface{} `json:"modules"`
}

func NewDebug() *Debug {
	return &Debug{
		make(map[string]interface{}),
	}
}

func (d *Debug) Append(name string, data interface{}) *Debug {
	d.Modules[name] = data
	return d
}

func GetDebugFromContext(ctx context.Context) (*Debug, bool) {
	if ctx == nil {
		return nil, false
	}
	ctxValue := ctx.Value(DebugContextKey)
	if ctxValue == nil {
		return nil, false
	}
	debugData, ok := ctxValue.(*Debug)
	if !ok {
		return nil, false
	}
	return debugData, true
}
