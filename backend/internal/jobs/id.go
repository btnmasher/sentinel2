package jobs

import (
	"crypto/rand"
	"encoding/hex"
)

func newJobID(prefix string) string {
	id := newID()
	if prefix == "" {
		return id
	}
	return prefix + ":" + id
}

func newID() string {
	buf := make([]byte, 16)
	if _, err := rand.Read(buf); err != nil {
		return "unknown"
	}
	return hex.EncodeToString(buf)
}
