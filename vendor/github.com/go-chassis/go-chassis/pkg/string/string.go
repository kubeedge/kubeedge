package stringutil

import (
	"strings"
	"unsafe"
)

// StringInSlice convert string to bool
func StringInSlice(a string, list []string) bool {
	for _, b := range list {
		if b == a {
			return true
		}
	}
	return false
}

// Str2bytes convert string to array of byte
func Str2bytes(s string) []byte {
	x := (*[2]uintptr)(unsafe.Pointer(&s))
	h := [3]uintptr{x[0], x[1], x[1]}
	return *(*[]byte)(unsafe.Pointer(&h))
}

// Bytes2str convert array of byte to string
func Bytes2str(b []byte) string {
	return *(*string)(unsafe.Pointer(&b))
}

// SplitToTwo split the string
func SplitToTwo(s, sep string) (string, string) {
	index := strings.Index(s, sep)
	if index < 0 {
		return "", s
	}
	return s[:index], s[index+len(sep):]
}

// SplitFirstSep split the string
func SplitFirstSep(s, sep string) string {
	index := strings.Index(s, sep)
	if index < 0 {
		return ""
	}
	return s[:index]
}

// MinInt check the minimum value of two integers
func MinInt(x, y int) int {
	if x <= y {
		return x
	}

	return y
}

// ClearStringMemory clear string memory, for very sensitive security related data
////you should clear it in memory after use
func ClearStringMemory(src *string) {
	p := (*struct {
		ptr uintptr
		len int
	})(unsafe.Pointer(src))

	len := MinInt(p.len, 32)
	ptr := p.ptr
	for idx := 0; idx < len; idx = idx + 1 {
		b := (*byte)(unsafe.Pointer(&ptr))
		*b = 0
		ptr++
	}
}

//ClearByteMemory clear byte memory, for very sensitive security related data
//you should clear it in memory after use
func ClearByteMemory(src []byte) {
	len := MinInt(len(src), 32)
	for idx := 0; idx < len; idx = idx + 1 {
		src[idx] = 0
	}
}
