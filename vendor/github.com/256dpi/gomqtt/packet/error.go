package packet

import "fmt"

// Error represents decoding and encoding errors.
type Error struct {
	Type Type

	format    string
	arguments []interface{}
}

func makeError(typ Type, format string, arguments ...interface{}) *Error {
	return &Error{Type: typ, format: format, arguments: arguments}
}

// Error implements the error interface.
func (e *Error) Error() string {
	return fmt.Sprintf(e.format, e.arguments...)
}
