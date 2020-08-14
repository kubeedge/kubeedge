package strutil

import (
	"crypto/md5"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"net/url"
	"strings"
	"text/template"
)

var (
	// EscapeJS escape javascript string
	EscapeJS = template.JSEscapeString
	// EscapeHTML escape html string
	EscapeHTML = template.HTMLEscapeString
)

// Md5 Generate a 32-bit md5 string
func Md5(src interface{}) string {
	return GenMd5(src)
}

// GenMd5 Generate a 32-bit md5 string
func GenMd5(src interface{}) string {
	h := md5.New()

	if s, ok := src.(string); ok {
		h.Write([]byte(s))
	} else {
		h.Write([]byte(fmt.Sprint(src)))
	}

	return hex.EncodeToString(h.Sum(nil))
}

// Base64 base64 encode
func Base64(str string) string {
	return base64.StdEncoding.EncodeToString([]byte(str))
}

// B64Encode base64 encode
func B64Encode(str string) string {
	return base64.StdEncoding.EncodeToString([]byte(str))
}

// URLEncode encode url string.
func URLEncode(s string) string {
	if pos := strings.IndexRune(s, '?'); pos > -1 { // escape query data
		return s[0:pos+1] + url.QueryEscape(s[pos+1:])
	}

	return s
}

// URLDecode decode url string.
func URLDecode(s string) string {
	if pos := strings.IndexRune(s, '?'); pos > -1 { // un-escape query data
		qy, err := url.QueryUnescape(s[pos+1:])
		if err == nil {
			return s[0:pos+1] + qy
		}
	}

	return s
}
