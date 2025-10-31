package grpc

import (
	"crypto/rand"
	"encoding/hex"
)

func generateSubscriberID() string {
	bytes := make([]byte, 8)
	rand.Read(bytes)
	return hex.EncodeToString(bytes)
}
