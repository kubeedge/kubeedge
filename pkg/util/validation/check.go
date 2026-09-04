package validation

import (
	"os"
)

// FileIsExist reports whether the file at the given path exists.
// It returns true when os.Stat succeeds, or when the stat error is anything
// other than "not exist" (e.g. permission denied still means the file is there).
func FileIsExist(path string) bool {
	_, err := os.Stat(path)
	return !os.IsNotExist(err)
}
