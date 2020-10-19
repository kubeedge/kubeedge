package mappercommon

import (
	"errors"
	"strconv"
)

// String to other types
func Convert(t string, value string) (r interface{}, err error) {
	switch t {
	case "int":
		return strconv.ParseInt(value, 10, 64)
	case "float":
		return strconv.ParseFloat(value, 32)
	case "double":
		return strconv.ParseFloat(value, 64)
	case "boolean":
		return strconv.ParseBool(value)
	case "string":
		return value, nil
	default:
		return nil, errors.New("Convert failed")
	}
}
