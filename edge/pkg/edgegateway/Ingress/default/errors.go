package defaults

import "github.com/pkg/errors"

var (
	// ErrMissingAnnotations the ingress rule does not contain annotations
	// This is an error only when annotations are being parsed
	ErrMissingAnnotations = errors.New("ingress rule without annotations")

	// ErrInvalidAnnotationName the ingress rule does contains an invalid
	// annotation name
	ErrInvalidAnnotationName = errors.New("invalid annotation name")
)

// LocationDenied error
type LocationDenied struct {
	Reason error
}

func (e LocationDenied) Error() string {
	return e.Reason.Error()
}

// IsMissingAnnotations checks if the err is an error which
// indicates the ingress does not contain annotations
func IsMissingAnnotations(e error) bool {
	return e == ErrMissingAnnotations
}

// IsLocationDenied checks if the err is an error which
// indicates a location should return HTTP code 503
func IsLocationDenied(e error) bool {
	_, ok := e.(LocationDenied)
	return ok
}

// NewLocationDenied returns a new LocationDenied error
func NewLocationDenied(reason string) error {
	return LocationDenied{
		Reason: errors.Errorf("Location denied, reason: %v", reason),
	}
}

// InvalidContent error
type InvalidContent struct {
	Name string
}

func (e InvalidContent) Error() string {
	return e.Name
}
