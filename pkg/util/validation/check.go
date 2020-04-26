package validation

import (
	"os"
	"strings"
)

// FileIsExist check file is exist
func FileIsExist(path string) bool {
	_, err := os.Stat(path)
	if err == nil {
		return true
	}
	return os.IsExist(err)
}

// to ensure the parent directory exist
func EnsureParentSubExist(path string) error {
	cadir := GetParentDirectory(path)
	if err := os.MkdirAll(cadir, 0666); err != nil {
		return err
	}
	return nil
}

func GetParentDirectory(dir string) string {
	return substr(dir, 0, strings.LastIndex(dir, "/"))
}

func substr(s string, pos, length int) string {
	runes := []rune(s)
	l := pos + length
	if l > len(runes) {
		l = len(runes)
	}
	return string(runes[pos:l])
}