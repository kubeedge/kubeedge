package packer

import "testing"

// TestPack is function to test Pack().
func TestPack(t *testing.T) {
	headerBuffer := make([]byte, 0)
	tests := []struct {
		name   string
		buffer *[]byte
	}{
		{
			name:   "PackTest",
			buffer: &headerBuffer,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := &PackageHeader{}
			h.Pack(tt.buffer)
		})
	}
}

// TestUnpack is function to test Unpack().
func TestUnpack(t *testing.T) {
	headerBuffer := make([]byte, 10)
	tests := []struct {
		name   string
		header []byte
	}{
		{
			name:   "UnpackTest",
			header: headerBuffer,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := &PackageHeader{}
			h.Unpack(tt.header)
		})
	}
}
