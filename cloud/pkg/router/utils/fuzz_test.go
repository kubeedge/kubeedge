package utils

import (
	"testing"
)

func FuzzRuleContains(f *testing.F) {
	f.Fuzz(func(t *testing.T, path1, path2 string) {
		RuleContains(path1, path2)
	})
}
