package common

// DataMethod defined standard model for deviceMethod
type DataMethod struct {
	Methods []Method
}

type Method struct {
	Name       string
	Path       string
	Parameters []Parameter
}

type Parameter struct {
	PropertyName string
	ValueType    string
}
