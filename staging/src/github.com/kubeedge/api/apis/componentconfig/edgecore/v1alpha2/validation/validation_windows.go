//go:build windows

package validation

import "k8s.io/apimachinery/pkg/util/validation/field"

// ValidateCgroupDriver validates `e` and returns an errorList if it is invalid
func ValidateCgroupDriver(cgroupDriver string) *field.Error {
	return nil
}
