package dtcommon

import (
	"errors"
	"regexp"
	"strconv"
	"strings"
)

//ValidateValue validate value type
func ValidateValue(valueType string, value string) error {
	switch valueType {
	case "":
		valueType = "string"
		return nil
	case "string":
		return nil
	case "int":
		_, err := strconv.ParseInt(value, 10, 64)
		if err != nil {
			return errors.New("the value is not int")
		}
		return nil
	case "float":
		_, err := strconv.ParseFloat(value, 64)
		if err != nil {
			return errors.New("the value is not float")
		}
		return nil
	case "boolean":
		if strings.Compare(value, "true") != 0 && strings.Compare(value, "false") != 0 {
			return errors.New("the bool value must be true or false")
		}
		return nil
	case "deleted":
		return nil
	default:
		return errors.New("the value type is not allowed")
	}
}

//ValidateTwinKey validate twin key
func ValidateTwinKey(key string) bool {
	pattern := "^[a-zA-Z0-9-_.,:/@#]{1,128}$"
	match, _ := regexp.MatchString(pattern, key)
	return match
}

//ValidateTwinValue validate twin value
func ValidateTwinValue(value string) bool {
	pattern := "^[a-zA-Z0-9-_.,:/@#]{1,512}$"
	match, _ := regexp.MatchString(pattern, value)
	return match
}
