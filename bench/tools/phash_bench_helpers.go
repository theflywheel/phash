package main

import (
	"crypto/rand"
	"fmt"
	"math/big"
	"runtime"
)

// GetMemoryUsage returns the current memory usage of the process in human-readable form
func GetMemoryUsage() string {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	return fmt.Sprintf("%.2f MB", float64(m.Alloc)/(1024*1024))
}

// GenerateUUID creates a random 16-byte UUID
func GenerateUUID() []byte {
	uuid := make([]byte, 16)
	_, err := rand.Read(uuid)
	if err != nil {
		panic(err)
	}
	// Set version (4) and variant (RFC4122)
	uuid[6] = (uuid[6] & 0x0F) | 0x40
	uuid[8] = (uuid[8] & 0x3F) | 0x80
	return uuid
}

// GenerateAlphanumeric creates a random alphanumeric string of given length
func GenerateAlphanumeric(length int) []byte {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	result := make([]byte, length)
	for i := 0; i < length; i++ {
		n, err := rand.Int(rand.Reader, big.NewInt(int64(len(charset))))
		if err != nil {
			panic(err)
		}
		result[i] = charset[n.Int64()]
	}
	return result
}
