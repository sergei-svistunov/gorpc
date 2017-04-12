package http_json

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
)

const (
	multipartMaxMemorySize = 1024 * 1024 * 512
)

type MultipartGetter struct {
	ParametersGetter
}

func NewMultipartGetter(req *http.Request) (*MultipartGetter, error) {
	if req == nil {
		return nil, fmt.Errorf("request is empty")
	}

	return &MultipartGetter{
		ParametersGetter: ParametersGetter{
			Req: req,
		},
	}, nil
}

// Add upload files to Params like a string
// params["{inputName}"] = string(file)
// params["filename_{inputName}"] = string(filename)
//
// Example:
// type MyData struct {
//     File     string `key:"my_file"`
//     Filename string `key:"filename_my_file"`
// }
func (g *MultipartGetter) Parse() error {
	if err := g.Req.ParseMultipartForm(multipartMaxMemorySize); err != nil {
		return fmt.Errorf("ParseMultipartForm %v", err)
	}

	if err := g.parseForm(); err != nil {
		return fmt.Errorf("parseForm %v", err)
	}

	for key, files := range g.Req.MultipartForm.File {
		if len(files) == 0 {
			continue
		}

		file := files[0]

		f, err := file.Open()
		if err != nil {
			return fmt.Errorf("file open %v", err)
		}

		b, err := ioutil.ReadAll(f)
		if err != nil {
			return fmt.Errorf("file readall %v", err)
		}

		key = strings.ToLower(key)
		g.values.Set(key, string(b))
		g.values.Set("filename_"+key, file.Filename)
	}

	return nil
}
