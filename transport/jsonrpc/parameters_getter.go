package jsonrpc

import (
	"encoding/json"
	"errors"
	"strconv"
)

type ParametersGetter struct {
	values interface{}
}

func (p *ParametersGetter) Parse() error {
	return nil
}

func (p *ParametersGetter) Fork(values map[string]interface{}) interface{} {
	return &ParametersGetter{values: values}
}

func (p *ParametersGetter) IsExists(path []string, name string) bool {
	v, ok := p.get(path, name)
	return ok && v != nil
}

func (p *ParametersGetter) GetString(path []string, name string) (string, error) {
	v, _ := p.get(path, name)
	if s, ok := v.(string); ok {
		return s, nil
	}
	return "", errors.New(`Wrong value of param "` + name + `". It must be string`)
}

func (p *ParametersGetter) GetBool(path []string, name string) (bool, error) {
	v, _ := p.get(path, name)
	if b, ok := v.(bool); ok {
		return b, nil
	}
	return false, errors.New(`Wrong value of param "` + name + `". It must be boolean`)
}

func (p *ParametersGetter) GetUint(path []string, name string) (uint, error) {
	v, err := p.GetInt64(path, name)
	return uint(v), err
}

func (p *ParametersGetter) GetByte(path []string, name string) (uint8, error) {
	return p.GetUint8(path, name)
}

func (p *ParametersGetter) GetUint8(path []string, name string) (uint8, error) {
	v, err := p.GetInt64(path, name)
	return uint8(v), err
}

func (p *ParametersGetter) GetUint16(path []string, name string) (uint16, error) {
	v, err := p.GetInt64(path, name)
	return uint16(v), err
}

func (p *ParametersGetter) GetUint32(path []string, name string) (uint32, error) {
	v, err := p.GetInt64(path, name)
	return uint32(v), err
}

func (p *ParametersGetter) GetUint64(path []string, name string) (uint64, error) {
	v, err := p.GetInt64(path, name)
	return uint64(v), err
}

func (p *ParametersGetter) GetInt(path []string, name string) (int, error) {
	v, err := p.GetInt64(path, name)
	return int(v), err
}

func (p *ParametersGetter) GetInt8(path []string, name string) (int8, error) {
	v, err := p.GetInt64(path, name)
	return int8(v), err
}

func (p *ParametersGetter) GetInt16(path []string, name string) (int16, error) {
	v, err := p.GetInt64(path, name)
	return int16(v), err
}

func (p *ParametersGetter) GetInt32(path []string, name string) (int32, error) {
	v, err := p.GetInt64(path, name)
	return int32(v), err
}

func (p *ParametersGetter) GetInt64(path []string, name string) (int64, error) {
	n, err := p.getNumber(path, name)
	if err != nil {
		return 0, err
	}
	return n.Int64()
}

func (p *ParametersGetter) GetFloat32(path []string, name string) (float32, error) {
	f, err := p.GetFloat64(path, name)
	return float32(f), err
}

func (p *ParametersGetter) GetFloat64(path []string, name string) (float64, error) {
	n, err := p.getNumber(path, name)
	if err != nil {
		return 0, err
	}
	return n.Float64()
}

func (p *ParametersGetter) getNumber(path []string, name string) (json.Number, error) {
	v, _ := p.get(path, name)
	if n, ok := v.(json.Number); ok {
		return n, nil
	}
	return json.Number(""), errors.New(`Wrong value of param "` + name + `". It must be a number`)
}

func (p *ParametersGetter) TraverseSlice(path []string, name string, h func(i int, v interface{}) error) (bool, error) {
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

func (p *ParametersGetter) TraverseMap(path []string, name string, h func(k string, v interface{}) error) (bool, error) {
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

func (p *ParametersGetter) get(path []string, name string) (interface{}, bool) {
	var m interface{}
	m = p.values

	for _, key := range append(path, name) {
		switch v := m.(type) {
		case map[string]interface{}:
			var exists bool
			m, exists = v[key]
			if !exists {
				return nil, false
			}
		case []interface{}:
			i, _ := strconv.ParseInt(key, 10, 64)
			m = v[i]
		default:
			return nil, false
		}
	}

	return m, true
}
