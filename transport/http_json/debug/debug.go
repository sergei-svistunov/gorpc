package debug

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
