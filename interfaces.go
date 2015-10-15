package gorpc

type IHandler interface {
	Caption() string
	Description() string
}

type IHandlerParameters interface {
	Parse() error

	IsExists([]string, string) bool

	GetString([]string, string) (string, error)

	GetBool([]string, string) (bool, error)

	GetUint([]string, string) (uint, error)
	GetByte([]string, string) (byte, error)
	GetUint8([]string, string) (uint8, error)
	GetUint16([]string, string) (uint16, error)
	GetUint32([]string, string) (uint32, error)
	GetUint64([]string, string) (uint64, error)

	GetInt([]string, string) (int, error)
	GetInt8([]string, string) (int8, error)
	GetInt16([]string, string) (int16, error)
	GetInt32([]string, string) (int32, error)
	GetInt64([]string, string) (int64, error)

	GetFloat32([]string, string) (float32, error)
	GetFloat64([]string, string) (float64, error)

	GetStringSlice([]string, string) []string
}
