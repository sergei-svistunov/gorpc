package http_json

import (
	"errors"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

type ParametersGetter struct {
	Req    *http.Request
	values url.Values
}

func (pg *ParametersGetter) Parse() error {
	return pg.parseForm()
}

func (pg *ParametersGetter) IsExists(path []string, name string) bool {
	return pg.get(name) != ""
}

func (pg *ParametersGetter) GetString(path []string, name string) (string, error) {
	return pg.get(name), nil
}

func (pg *ParametersGetter) GetBool(path []string, name string) (bool, error) {
	v, err := strconv.ParseBool(pg.get(name))
	if err != nil {
		err = errors.New(`Wrong value of param "` + name + `". It should be Bool`)
	}
	return v, err
}

func (pg *ParametersGetter) GetUint(path []string, name string) (uint, error) {
	v, err := strconv.ParseUint(pg.get(name), 0, 0)
	if err != nil {
		err = errors.New(`Wrong value of param "` + name + `". It should be Uint`)
	}
	return uint(v), err
}

func (pg *ParametersGetter) GetByte(path []string, name string) (uint8, error) {
	return pg.GetUint8(path, name)
}

func (pg *ParametersGetter) GetUint8(path []string, name string) (uint8, error) {
	v, err := strconv.ParseUint(pg.get(name), 0, 8)
	if err != nil {
		err = errors.New(`Wrong value of param "` + name + `". It should be Uint`)
	}
	return uint8(v), err
}

func (pg *ParametersGetter) GetUint16(path []string, name string) (uint16, error) {
	v, err := strconv.ParseUint(pg.get(name), 0, 16)
	if err != nil {
		err = errors.New(`Wrong value of param "` + name + `". It should be Uint`)
	}
	return uint16(v), err
}

func (pg *ParametersGetter) GetUint32(path []string, name string) (uint32, error) {
	v, err := strconv.ParseUint(pg.get(name), 0, 32)
	if err != nil {
		err = errors.New(`Wrong value of param "` + name + `". It should be Uint`)
	}
	return uint32(v), err
}

func (pg *ParametersGetter) GetUint64(path []string, name string) (uint64, error) {
	v, err := strconv.ParseUint(pg.get(name), 0, 64)
	if err != nil {
		err = errors.New(`Wrong value of param "` + name + `". It should be Uint`)
	}
	return v, err
}

func (pg *ParametersGetter) GetInt(path []string, name string) (int, error) {
	v, err := strconv.ParseInt(pg.get(name), 0, 0)
	if err != nil {
		err = errors.New(`Wrong value of param "` + name + `". It should be Int`)
	}
	return int(v), err
}

func (pg *ParametersGetter) GetInt8(path []string, name string) (int8, error) {
	v, err := strconv.ParseInt(pg.get(name), 0, 8)
	if err != nil {
		err = errors.New(`Wrong value of param "` + name + `". It should be Int`)
	}
	return int8(v), err
}

func (pg *ParametersGetter) GetInt16(path []string, name string) (int16, error) {
	v, err := strconv.ParseInt(pg.get(name), 0, 16)
	if err != nil {
		err = errors.New(`Wrong value of param "` + name + `". It should be Int`)
	}
	return int16(v), err
}

func (pg *ParametersGetter) GetInt32(path []string, name string) (int32, error) {
	v, err := strconv.ParseInt(pg.get(name), 0, 32)
	if err != nil {
		err = errors.New(`Wrong value of param "` + name + `". It should be Int`)
	}
	return int32(v), err
}

func (pg *ParametersGetter) GetInt64(path []string, name string) (int64, error) {
	v, err := strconv.ParseInt(pg.get(name), 0, 64)
	if err != nil {
		err = errors.New(`Wrong value of param "` + name + `". It should be Int`)
	}
	return v, err
}

func (pg *ParametersGetter) GetFloat32(path []string, name string) (float32, error) {
	v, err := strconv.ParseFloat(pg.get(name), 32)
	if err != nil {
		err = errors.New(`Wrong value of param "` + name + `". It should be Float`)
	}
	return float32(v), err
}

func (pg *ParametersGetter) GetFloat64(path []string, name string) (float64, error) {
	v, err := strconv.ParseFloat(pg.get(name), 64)
	if err != nil {
		err = errors.New(`Wrong value of param "` + name + `". It should be Float`)
	}
	return v, err
}

func (pg *ParametersGetter) GetStringSlice(path []string, name string) []string {
	return pg.getSlice(name)
}

func (pg *ParametersGetter) get(name string) string {
	slice := pg.getSlice(name)
	if len(slice) == 0 {
		return ""
	}
	return slice[0]
}

func (pg *ParametersGetter) getSlice(name string) []string {
	name = strings.ToLower(name)
	if slice, ok := pg.values[name]; ok {
		return slice
	}
	return []string{}
}

func (pg *ParametersGetter) parseForm() (err error) {
	if pg.values != nil {
		return
	}
	if err = pg.Req.ParseForm(); err != nil {
		return
	}
	pg.values = make(url.Values)
	for k, vs := range pg.Req.Form {
		name := strings.ToLower(k)
		for _, v := range vs {
			pg.values.Add(name, v)
		}
	}
	return
}
