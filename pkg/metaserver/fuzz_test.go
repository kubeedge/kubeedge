package metaserver

import (
	"testing"
)

func FuzzParseKey(f *testing.F) {
	f.Fuzz(func(t *testing.T, data string) {
		_, _, _ = ParseKey(data)
	})
}
