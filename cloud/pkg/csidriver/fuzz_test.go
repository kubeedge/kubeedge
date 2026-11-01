package csidriver

import (
	"testing"
)

func FuzzExtractMessage(f *testing.F) {
	f.Fuzz(func(t *testing.T, data []byte) {
		result, err := extractMessage(string(data))
		if err == nil {
			_ = result.GetContent().(string)
			_ = result.GetOperation()
		}
	})
}
