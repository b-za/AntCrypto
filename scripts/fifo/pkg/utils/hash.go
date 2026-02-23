package utils

import (
	"crypto/md5"
	"encoding/hex"
)

func GenerateHash(input string) string {
	hash := md5.Sum([]byte(input))
	return hex.EncodeToString(hash[:])
}

func TruncateHash(hash string, length int) string {
	if len(hash) < length {
		return hash
	}
	return hash[:length]
}
