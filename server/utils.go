package server

import (
	"crypto/rand"
	"encoding/hex"
)

func generateID() string {
	bytes := make([]byte, 3)
	rand.Read(bytes)
	return hex.EncodeToString(bytes)
}
