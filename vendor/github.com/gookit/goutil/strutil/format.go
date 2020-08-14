package strutil

import (
	"regexp"
	"strings"
)

// Some alias methods.
var (
	Lower = strings.ToLower
	Upper = strings.ToUpper
	Title = strings.ToTitle
)

/*************************************************************
 * change string case
 *************************************************************/

// Lowercase alias of the strings.ToLower()
func Lowercase(s string) string {
	return strings.ToLower(s)
}

// Uppercase alias of the strings.ToUpper()
func Uppercase(s string) string {
	return strings.ToUpper(s)
}

// UpperWord Change the first character of each word to uppercase
func UpperWord(s string) string {
	if len(s) == 0 {
		return s
	}

	ss := strings.Split(s, " ")
	ns := make([]string, len(ss))
	for i, word := range ss {
		ns[i] = UpperFirst(word)
	}
	return strings.Join(ns, " ")
}

// LowerFirst lower first char
func LowerFirst(s string) string {
	if len(s) == 0 {
		return s
	}

	f := s[0]
	if f >= 'A' && f <= 'Z' {
		return strings.ToLower(string(f)) + s[1:]
	}
	return s
}

// UpperFirst upper first char
func UpperFirst(s string) string {
	if len(s) == 0 {
		return s
	}

	f := s[0]
	if f >= 'a' && f <= 'z' {
		return strings.ToUpper(string(f)) + s[1:]
	}
	return s
}

// Snake alias of the SnakeCase
func Snake(s string, sep ...string) string {
	return SnakeCase(s, sep...)
}

// SnakeCase convert. eg "RangePrice" -> "range_price"
func SnakeCase(s string, sep ...string) string {
	sepChar := "_"
	if len(sep) > 0 {
		sepChar = sep[0]
	}

	newStr := toSnakeReg.ReplaceAllStringFunc(s, func(s string) string {
		return sepChar + LowerFirst(s)
	})

	return strings.TrimLeft(newStr, sepChar)
}

// Camel alias of the CamelCase
func Camel(s string, sep ...string) string {
	return CamelCase(s, sep...)
}

// CamelCase convert string to camel case.
// Support:
// 	"range_price" -> "rangePrice"
// 	"range price" -> "rangePrice"
// 	"range-price" -> "rangePrice"
func CamelCase(s string, sep ...string) string {
	sepChar := "_"
	if len(sep) > 0 {
		sepChar = sep[0]
	}

	// Not contains sep char
	if !strings.Contains(s, sepChar) {
		return s
	}

	// Get regexp instance
	rgx, ok := toCamelRegs[sepChar]
	if !ok {
		rgx = regexp.MustCompile(regexp.QuoteMeta(sepChar) + "+[a-zA-Z]")
	}

	return rgx.ReplaceAllStringFunc(s, func(s string) string {
		s = strings.TrimLeft(s, sepChar)
		return UpperFirst(s)
	})
}
