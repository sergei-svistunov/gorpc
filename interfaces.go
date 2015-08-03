package gorpc

import (
	"time"
)

type IHandlerParameters interface {
	IsExists(string) bool

	GetString(string) (string, error)

	GetBool(string) (bool, error)

	GetUint(string) (uint, error)
	GetByte(string) (byte, error)
	GetUint8(string) (uint8, error)
	GetUint16(string) (uint16, error)
	GetUint32(string) (uint32, error)
	GetUint64(string) (uint64, error)

	GetInt(string) (int, error)
	GetInt8(string) (int8, error)
	GetInt16(string) (int16, error)
	GetInt32(string) (int32, error)
	GetInt64(string) (int64, error)

	GetFloat32(string) (float32, error)
	GetFloat64(string) (float64, error)

	GetStringSlice(string) []string
}

type ICache interface {
	Get(key string) (interface{}, bool)
	Put(key string, data interface{}, ttl time.Duration)
}
