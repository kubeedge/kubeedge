package util

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// GetCurrentDirectory get dir
func GetCurrentDirectory() string {
	dir, err := filepath.Abs(filepath.Dir(os.Args[0]))
	if err != nil {
		fmt.Printf("error when reading currentDirectory:%v\n", err)
	}
	return strings.Replace(dir, "\\", "/", -1)
}
