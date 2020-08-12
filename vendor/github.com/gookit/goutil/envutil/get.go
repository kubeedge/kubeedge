package envutil

import "os"

// Getenv get ENV value by key name
func Getenv(name string, def ...string) string {
	val := os.Getenv(name)
	if val == "" && len(def) > 0 {
		val = def[0]
	}

	return val
}
