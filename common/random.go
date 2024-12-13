package common

import (
	"crypto/rand"
	"encoding/hex"
)

func GenerateRandomHex(len int) string {
	bytes := make([]byte, len)
	_, _ = rand.Read(bytes)
	return hex.EncodeToString(bytes)
}
