# String Util

This is an go string operate util package.

- Github: https://github.com/gookit/goutil/strutil
- GoDoc: https://godoc.org/github.com/gookit/goutil/strutil

## Install

```bash
go get github.com/gookit/goutil/dump
```

## Usage

```go
ss := strutil.ToArray("a,b,c", ",")
// Output: []string{"a", "b", "c"}

ints, err := strutil.ToIntSlice("1,2,3")
// Output: []int{1, 2, 3}
```

## Functions

```text
func B64Encode(str string) string
func Bool(s string) (bool, error)
func Camel(s string, sep ...string) string
func CamelCase(s string, sep ...string) string
func FilterEmail(s string) string
func GenMd5(src interface{}) string
func LowerFirst(s string) string
func Lowercase(s string) string
func Md5(src interface{}) string
func MustBool(s string) bool
func MustString(in interface{}) string
func PadLeft(s, pad string, length int) string
func PadRight(s, pad string, length int) string
func Padding(s, pad string, length int, pos uint8) string
func PrettyJSON(v interface{}) (string, error)
func RandomBytes(length int) ([]byte, error)
func RandomString(length int) (string, error)
func RenderTemplate(input string, data interface{}, fns template.FuncMap, isFile ...bool) string
func Repeat(s string, times int) string
func RepeatRune(char rune, times int) (chars []rune)
func Replaces(str string, pairs map[string]string) string
func Similarity(s, t string, rate float32) (float32, bool)
func Snake(s string, sep ...string) string
func SnakeCase(s string, sep ...string) string
func Split(s, sep string) (ss []string)
func String(val interface{}) (string, error)
func Substr(s string, pos, length int) string
func ToArray(s string, sep ...string) []string
func ToBool(s string) (bool, error)
func ToIntSlice(s string, sep ...string) (ints []int, err error)
func ToInts(s string, sep ...string) ([]int, error)
func ToSlice(s string, sep ...string) []string
func ToString(val interface{}) (str string, err error)
func ToTime(s string, layouts ...string) (t time.Time, err error)
func Trim(s string, cutSet ...string) string
func TrimLeft(s string, cutSet ...string) string
func TrimRight(s string, cutSet ...string) string
func URLDecode(s string) string
func URLEncode(s string) string
func UpperFirst(s string) string
func UpperWord(s string) string
func Uppercase(s string) string
```
