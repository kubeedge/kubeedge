//go:build !windows

package validation

import (
	"k8s.io/apimachinery/pkg/util/validation/field"

	"github.com/kubeedge/api/apis/componentconfig/edgecore/v1alpha2"
)

// ValidateCgroupDriver validates `edged.TailoredKubeletConfig.CgroupDriver` and returns an errorList if it is invalid
func ValidateCgroupDriver(cgroupDriver string) *field.Error {
	switch cgroupDriver {
	case v1alpha2.CGroupDriverCGroupFS, v1alpha2.CGroupDriverSystemd:
	default:
		return field.Invalid(field.NewPath("CGroupDriver"), cgroupDriver,
			"CGroupDriver value error")
	}
	return nil
}
