package http_json

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
)

type JsonParametersGetter struct {
	Req    *http.Request
	values map[string]interface{}
}

func (p *JsonParametersGetter) Parse() error {
	defer p.Req.Body.Close()
	maxFormSize := int64(10 << 20) // 10 MB is a lot of text.
	reader := io.LimitReader(p.Req.Body, maxFormSize+1)
	decoder := json.NewDecoder(reader)
	decoder.UseNumber()
	return decoder.Decode(&p.values)
}

func (p *JsonParametersGetter) IsExists(path []string, name string) bool {
	_, ok := p.get(path, name)
	return ok
}

func (p *JsonParametersGetter) GetString(path []string, name string) (string, error) {
	v, _ := p.get(path, name)
	if s, ok := v.(string); ok {
		return s, nil
	}
	return "", errors.New(`Wrong value of param "` + name + `". It must be string`)
}

func (p *JsonParametersGetter) GetBool(path []string, name string) (bool, error) {
	v, _ := p.get(path, name)
	if b, ok := v.(bool); ok {
		return b, nil
	}
	return false, errors.New(`Wrong value of param "` + name + `". It must be boolean`)
}

func (p *JsonParametersGetter) GetUint(path []string, name string) (uint, error) {
	v, err := p.GetInt64(path, name)
	return uint(v), err
}

func (p *JsonParametersGetter) GetByte(path []string, name string) (uint8, error) {
	return p.GetUint8(path, name)
}

func (p *JsonParametersGetter) GetUint8(path []string, name string) (uint8, error) {
	v, err := p.GetInt64(path, name)
	return uint8(v), err
}

func (p *JsonParametersGetter) GetUint16(path []string, name string) (uint16, error) {
	v, err := p.GetInt64(path, name)
	return uint16(v), err
}

func (p *JsonParametersGetter) GetUint32(path []string, name string) (uint32, error) {
	v, err := p.GetInt64(path, name)
	return uint32(v), err
}

func (p *JsonParametersGetter) GetUint64(path []string, name string) (uint64, error) {
	v, err := p.GetInt64(path, name)
	return uint64(v), err
}

func (p *JsonParametersGetter) GetInt(path []string, name string) (int, error) {
	v, err := p.GetInt64(path, name)
	return int(v), err
}

func (p *JsonParametersGetter) GetInt8(path []string, name string) (int8, error) {
	v, err := p.GetInt64(path, name)
	return int8(v), err
}

func (p *JsonParametersGetter) GetInt16(path []string, name string) (int16, error) {
	v, err := p.GetInt64(path, name)
	return int16(v), err
}

func (p *JsonParametersGetter) GetInt32(path []string, name string) (int32, error) {
	v, err := p.GetInt64(path, name)
	return int32(v), err
}

func (p *JsonParametersGetter) GetInt64(path []string, name string) (int64, error) {
	n, err := p.getNumber(path, name)
	if err != nil {
		return 0, err
	}
	return n.Int64()
}

func (p *JsonParametersGetter) GetFloat32(path []string, name string) (float32, error) {
	f, err := p.GetFloat64(path, name)
	return float32(f), err
}

func (p *JsonParametersGetter) GetFloat64(path []string, name string) (float64, error) {
	n, err := p.getNumber(path, name)
	if err != nil {
		return 0, err
	}
	return n.Float64()
}

func (p *JsonParametersGetter) getNumber(path []string, name string) (json.Number, error) {
	v, _ := p.get(path, name)
	if n, ok := v.(json.Number); ok {
		return n, nil
	}
	return json.Number(""), errors.New(`Wrong value of param "` + name + `". It must be number`)
}

func (p *JsonParametersGetter) GetStringSlice(path []string, name string) []string {
	v, _ := p.get(path, name)
	if a, ok := v.([]interface{}); ok {
		sa := make([]string, len(a))
		for i, e := range a {
			sa[i] = e.(string)
		}
		return sa
	}
	return []string{}
}

func (p *JsonParametersGetter) get(path []string, name string) (interface{}, bool) {
	m := p.values
	for _, key := range path {
		if v, ok := m[key]; ok {
			if m, ok = v.(map[string]interface{}); !ok {
				return nil, false
			}
		} else {
			return nil, false
		}
	}
	if v, ok := m[name]; ok {
		return v, true
	}
	return nil, false
}