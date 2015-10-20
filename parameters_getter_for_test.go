package gorpc

import (
	"bytes"
	"encoding/json"
	"errors"
	"strconv"
	"strings"
)

type ParametersGetter struct {
	Values map[string][]string
}

func (pg *ParametersGetter) Fork(m map[string]interface{}) interface{} {
	panic("not supported")
}

func (pg *ParametersGetter) Parse() error {
	return nil
}

func (pg *ParametersGetter) IsExists(path []string, name string) bool {
	_, exists := pg.Values[name]

	return exists
}

func (pg *ParametersGetter) TraverseSlice(path []string, name string, h func(i int, v interface{}) error) (bool, error) {
	name = strings.ToLower(name)
	if strSlice, ok := pg.Values[name]; ok {
		for i, v := range strSlice {
			if err := h(i, v); err != nil {
				return false, err
			}
		}
		return true, nil
	}
	return false, nil
}

func (pg *ParametersGetter) TraverseMap(path []string, name string, h func(k string, v interface{}) error) (bool, error) {
	panic("maps not supported")
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
	v, err := strconv.ParseUint(pg.get(name), 0, 8)
	if err != nil {
		err = errors.New(`Wrong value of param "` + name + `". It should be Uint`)
	}

	return uint8(v), err
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

func (pg *ParametersGetter) get(name string) string {
	slice := pg.Values[name]

	return slice[0]
}

type JsonParametersGetter struct {
	Req         string
	MaxFormSize int64
	values      map[string]interface{}
}

func (p *JsonParametersGetter) Parse() error {
	reader := bytes.NewBufferString(p.Req)
	decoder := json.NewDecoder(reader)
	decoder.UseNumber()
	return decoder.Decode(&p.values)
}

func (p *JsonParametersGetter) Fork(values map[string]interface{}) interface{} {
	return &JsonParametersGetter{values: values}
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

func (p *JsonParametersGetter) TraverseSlice(path []string, name string, h func(i int, v interface{}) error) (bool, error) {
	v, _ := p.get(path, name)
	if a, ok := v.([]interface{}); ok {
		origValues := p.values
		defer func() {
			p.values = origValues
		}()
		for i, v := range a {
			if m, ok := v.(map[string]interface{}); ok {
				p.values = m
			}
			if err := h(i, v); err != nil {
				return false, err
			}
		}
		return true, nil
	}
	return false, nil
}

func (p *JsonParametersGetter) TraverseMap(path []string, name string, h func(k string, v interface{}) error) (bool, error) {
	v, _ := p.get(path, name)
	if m, ok := v.(map[string]interface{}); ok {
		origValues := p.values
		defer func() {
			p.values = origValues
		}()
		for k, v := range m {
			if submap, ok := v.(map[string]interface{}); ok {
				p.values = submap
			}
			if err := h(k, v); err != nil {
				return false, err
			}
		}
		return true, nil
	}
	return false, nil
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
