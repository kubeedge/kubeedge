package stream

import (
	"bytes"
	"testing"
)

func FuzzReadMessageFromTunnel(f *testing.F) {
	f.Fuzz(func(t *testing.T, data []byte) {
		_, _ = ReadMessageFromTunnel(bytes.NewReader(data))
	})
}
