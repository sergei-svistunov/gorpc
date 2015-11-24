package http_json

import (
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
)

func etagHash(response interface{}) (string, error) {
	b, err := json.Marshal(response)
	if err != nil {
		return "", err
	}
	hash := sha1.Sum(b)
	return hex.EncodeToString(hash[:]), nil
}
