package swagger

import (
	"regexp"
)

func getModelName(name string) string {
	reg := regexp.MustCompile("[a-z]+\\.")
	modelName := reg.ReplaceAllString(name, "")
	modelName = "#/definitions/" + modelName
	return modelName
}

func getOtherName(name string) string {
	switch name {
	case "uint", "uint8", "uint16", "uint32", "uint64", "int", "int8", "int16", "int32", "int64", "byte":
		return "integer"
	case "float32", "float64":
		return "number"
	case "time.Time":
		return "string"
	case "bool":
		return "boolean"
	default:
		return name
	}
}

func getFormat(name string) string {
	switch name {
	case "int64":
		return "int64"
	case "int", "int32":
		return "int32"
	case "float32":
		return "float"
	case "float64":
		return "double"
	case "byte", "uint8":
		return "byte"
	case "time.Time", "*time.Time", "datetime", "dateTime":
		return "date-time"
	default:
		return ""
	}
}