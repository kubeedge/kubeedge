package udsserver

import (
	"testing"
)

func FuzzExtractMessage(f *testing.F) {
	f.Fuzz(func(t *testing.T, data string) {
		msg, err := ExtractMessage(data)
		if err != nil {
			_ = feedbackError(err, msg)
		}
	})
}
