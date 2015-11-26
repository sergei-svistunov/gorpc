package cache

import (
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
)

func ETagHash(response interface{}) (string, error) {
	b, err := json.Marshal(response)
	if err != nil {
		return "", err
	}
	hash := sha1.Sum(b)
	return hex.EncodeToString(hash[:]), nil
}
