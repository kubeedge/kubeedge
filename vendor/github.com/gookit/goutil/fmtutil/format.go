package fmtutil

import (
	"encoding/json"
	"fmt"

	"github.com/gookit/goutil/arrutil"
)

// DataSize format bytes number friendly.
// Usage:
// 	file, err := os.Open(path)
// 	fl, err := file.Stat()
// 	fmtSize := DataSize(fl.Size())
func DataSize(bytes uint64) string {
	switch {
	case bytes < 1024:
		return fmt.Sprintf("%dB", bytes)
	case bytes < 1024*1024:
		return fmt.Sprintf("%.2fK", float64(bytes)/1024)
	case bytes < 1024*1024*1024:
		return fmt.Sprintf("%.2fM", float64(bytes)/1024/1024)
	default:
		return fmt.Sprintf("%.2fG", float64(bytes)/1024/1024/1024)
	}
}

// PrettyJSON get pretty Json string
func PrettyJSON(v interface{}) (string, error) {
	out, err := json.MarshalIndent(v, "", "    ")
	return string(out), err
}

// StringsToInts string slice to int slice. alias of the arrutil.StringsToInts()
func StringsToInts(ss []string) (ints []int, err error) {
	return arrutil.StringsToInts(ss)
}
