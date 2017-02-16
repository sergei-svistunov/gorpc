package http_json

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"strconv"
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

func (g *MultipartGetter) Parse() error {
	if err := g.Req.ParseMultipartForm(multipartMaxMemorySize); err != nil {
		return err
	}

	if err := g.parseForm(); err != nil {
		return err
	}

	for key, files := range g.Req.MultipartForm.File {
		if len(files) == 0 {
			continue
		}

		file := files[0]

		f, err := file.Open()
		if err != nil {
			return err
		}

		b, err := ioutil.ReadAll(f)
		if err != nil {
			return err
		}

		key = strings.ToLower(key)
		for i := range b {
			g.values.Add(key, strconv.FormatUint(uint64(b[i]), 10))
		}
	}

	return nil
}
