package debug

import (
	"context"
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
	if debugData, ok := ctxValue.(*Debug); ok {
		return debugData, true
	}
	return nil, false
}

func Add(ctx context.Context, name string, data interface{}) bool {
	if debug, isEnable := GetDebugFromContext(ctx); isEnable {
		debug.Append(name, data)
		return true
	}
	return false
}

func IsOn(ctx context.Context) bool {
	_, isEnable := GetDebugFromContext(ctx)
	return isEnable
}
