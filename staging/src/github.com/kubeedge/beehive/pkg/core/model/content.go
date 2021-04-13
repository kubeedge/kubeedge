package model

import (
	"encoding/json"
	"fmt"
	"strconv"
	"unsafe"
)

// Content the is various types of Content conversion
type Content interface {
	fmt.Stringer
	json.Marshaler
	GetString() (string, bool)
	GetBytes() ([]byte, bool)
	Raw() interface{}
}

func NewContent(val interface{}) Content {
	if val == nil {
		return rawContent{}
	}
	switch v := val.(type) {
	case Content:
		return v
	case string:
		return stringContent(v)
	case []byte:
		return bytesContent(v)
	default:
		return rawContent{
			raw: v,
		}
	}
}

type stringContent string

func (s stringContent) String() string {
	return string(s)
}

func (s stringContent) GetString() (string, bool) {
	return string(s), true
}

func (s stringContent) GetBytes() ([]byte, bool) {
	tmp := string(s)
	conv := (*[]byte)(unsafe.Pointer(&tmp))
	return *conv, true
}

func (s stringContent) Raw() interface{} {
	return string(s)
}

func (s stringContent) MarshalJSON() ([]byte, error) {
	return json.Marshal(string(s))
}

type bytesContent []byte

func (b bytesContent) String() string {
	s, _ := b.GetString()
	return strconv.Quote(s)
}

func (b bytesContent) GetString() (string, bool) {
	tmp := []byte(b)
	conv := (*string)(unsafe.Pointer(&tmp))
	return *conv, true
}

func (b bytesContent) GetBytes() ([]byte, bool) {
	return []byte(b), true
}

func (b bytesContent) Raw() interface{} {
	return []byte(b)
}

func (b bytesContent) MarshalJSON() ([]byte, error) {
	// The bytes data will be passed after base64 encoding by default,
	// here it is fixed to not use base64 encoding
	s, _ := b.GetString()
	return json.Marshal(s)
}

type rawContent struct {
	raw interface{}
}

func (r rawContent) String() string {
	return fmt.Sprint(r.raw)
}

func (r rawContent) GetString() (string, bool) {
	return "", false
}

func (r rawContent) GetBytes() ([]byte, bool) {
	return nil, false
}

func (r rawContent) Raw() interface{} {
	return r.raw
}

func (r rawContent) MarshalJSON() ([]byte, error) {
	return json.Marshal(r.raw)
}
