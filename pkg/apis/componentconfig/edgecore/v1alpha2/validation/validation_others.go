//go:build !windows

package validation

import (
	"k8s.io/apimachinery/pkg/util/validation/field"
	"k8s.io/klog/v2"
	"k8s.io/kubernetes/pkg/apis/core/validation"

	"github.com/kubeedge/kubeedge/pkg/apis/componentconfig/edgecore/v1alpha2"
)

// ValidateModuleEdged validates `e` and returns an errorList if it is invalid
func ValidateModuleEdged(e v1alpha2.Edged) field.ErrorList {
	if !e.Enable {
		return field.ErrorList{}
	}
	allErrs := field.ErrorList{}
	messages := validation.ValidateNodeName(e.HostnameOverride, false)
	for _, msg := range messages {
		allErrs = append(allErrs, field.Invalid(field.NewPath("HostnameOverride"), e.HostnameOverride, msg))
	}
	if e.NodeIP == "" {
		klog.Warningf("NodeIP is empty , use default ip which can connect to cloud.")
	}
	switch e.TailoredKubeletConfig.CgroupDriver {
	case v1alpha2.CGroupDriverCGroupFS, v1alpha2.CGroupDriverSystemd:
	default:
		allErrs = append(allErrs, field.Invalid(field.NewPath("CGroupDriver"), e.TailoredKubeletConfig.CgroupDriver,
			"CGroupDriver value error"))
	}
	return allErrs
}
