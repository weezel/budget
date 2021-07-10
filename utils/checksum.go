package utils

import (
	"crypto/sha256"
	"encoding/hex"
)

func CalcSha256Sum(data []byte) string {
	if len(data) == 0 {
		return ""
	}
	tmpHash := sha256.Sum256(data)
	return hex.EncodeToString(tmpHash[:])[0:16]
}
