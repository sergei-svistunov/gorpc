package gorpc

import (
	"fmt"
	"strings"
)

type handlerInfo struct {
	Handler                         *handlerEntity
	Path, Caption, Description, Tag string
	Versions                        []handlerVersion
}

func (hi *handlerInfo) String() string {
	res := fmt.Sprintln(hi.Path) +
		fmt.Sprintf("\tCaption: %s\n\tDescription:\n\t\t%s\n\tVersions:\n",
			hi.Caption,
			strings.Replace(hi.Description, "\n", "\n\t\t", -1))
	for vN, version := range hi.Versions {
		res += fmt.Sprintf("\t\t%d:\n\t\t\tParameters:\n", vN+1)
		for _, parameter := range version.Parameters {
			res += fmt.Sprintf("\t\t\t\t%s:\n\t\t\t\t\tType: %s\n\t\t\t\t\tDescription: %s\n\t\t\t\t\tIs required: %t\n",
				parameter.Name, parameter.RawType.Kind().String(), parameter.Description, parameter.IsRequired)
		}
	}

	return res
}
