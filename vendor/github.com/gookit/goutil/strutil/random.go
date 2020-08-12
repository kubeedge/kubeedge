package strutil

import (
	"crypto/rand"
	"encoding/base64"
)

// RandomBytes generate
func RandomBytes(length int) ([]byte, error) {
	b := make([]byte, length)
	_, err := rand.Read(b)
	// Note that err == nil only if we read len(b) bytes.
	if err != nil {
		return nil, err
	}

	return b, nil
}

// RandomString generate.
// Example:
// this will give us a 44 byte, base64 encoded output
// 	token, err := RandomString(32)
// 	if err != nil {
//     // Serve an appropriately vague error to the
//     // user, but log the details internally.
// 	}
func RandomString(length int) (string, error) {
	b, err := RandomBytes(length)
	return base64.URLEncoding.EncodeToString(b), err
}
