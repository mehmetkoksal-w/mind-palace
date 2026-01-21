package contracts

import (
	"crypto/rand"
	"encoding/hex"
)

// GenerateID generates a unique ID with the given prefix.
func GenerateID(prefix string) string {
	b := make([]byte, 8)
	_, _ = rand.Read(b)
	return prefix + "_" + hex.EncodeToString(b)
}
