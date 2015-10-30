package adapter

import (
	"sync"
    "strings"
)

var (
	components []Component
	locker     sync.Mutex
)

type Component struct {
    Placeholder string
    Imports     []string
    code        string
    preHook     func(*string) error
}

func NewComponent(placeholder string, imports []string, code string, preHook func(*string) error) Component {
    return Component{
        placeholder,
        imports,
        code,
        preHook,
    }
}

func (c Component) GetCode() string {
    if c.preHook != nil {
        if err := c.preHook(&c.code); err != nil {
            // TODO log error
            return ""
        }
    }
    return c.code
}

func RegisterComponent(c Component) {
	locker.Lock()
	defer locker.Unlock()
	components = append(components, c)
}

func getComponents() []Component {
	locker.Lock()
	defer locker.Unlock()
	return components
}

func getComponentByPlaceholder(ph string) *Component {
	locker.Lock()
	defer locker.Unlock()
	for i := range components {
		if components[i].Placeholder == ph {
			return &components[i]
		}
	}
	return nil
}

func CollectImports(extraImports []string) string {
    var imports []string
	for _, comp := range getComponents() {
		if len(comp.Imports) > 0 {
			imports = append(imports, comp.Imports...)
		}
	}
    if len(extraImports) > 0 {
        imports = append(imports, extraImports...)
    }

	// TODO filter same imports here or just execute goimports on result

	var result string
	for i := range imports {
        if strings.HasSuffix(imports[i], "\"") {
            result += (imports[i] + "\n")
        } else {
            result += ("\"" + imports[i] + "\"\n")
        }
	}

	return result
}
