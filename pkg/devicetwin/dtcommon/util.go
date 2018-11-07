package dtcommon

import (
	"errors"
	"regexp"
	"strconv"
	"strings"
)

const (
	//MAXTWINNUM max twin key
	MAXTWINNUM = 64
)

//ValidateValue validate value type
func ValidateValue(valueType string, value string) error {
	if valueType == "" {
		valueType = "string"
	}
	if strings.Compare(valueType, "string") == 0 {
		return nil
	} else if strings.Compare(valueType, "int") == 0 {
		_, err := strconv.ParseInt(value, 10, 64)
		if err != nil {
			return errors.New("The value is not int")
		}
	} else if strings.Compare(valueType, "float") == 0 {
		_, err := strconv.ParseFloat(value, 64)
		if err != nil {
			return errors.New("The value is not float")
		}
	} else if strings.Compare(valueType, "boolean") == 0 {
		if strings.Compare(value, "true") == 0 || strings.Compare(value, "false") == 0 {
			return errors.New("The bool type must be true or false")
		}
	} else if strings.Compare(valueType, "deleted") == 0 {
		return nil
	} else {
		return errors.New("The value type is not allowed")
	}
	return nil
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
