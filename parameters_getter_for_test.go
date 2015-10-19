package gorpc

import (
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

func (pg *ParametersGetter) GetSlice(path []string, name string) []interface{} {
	name = strings.ToLower(name)
	if strSlice, ok := pg.Values[name]; ok {
		slice := make([]interface{}, len(strSlice))
		for i, v := range strSlice {
			slice[i] = v
		}
		return slice
	}
	return nil
}

func (pg *ParametersGetter) GetMap(path []string, name string) map[string]interface{} {
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
