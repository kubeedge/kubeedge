package validation

import (
	"k8s.io/apimachinery/pkg/util/validation/field"

	"github.com/kubeedge/kubeedge/edge/pkg/apis/edgecore/config"
)

func ValidateEdgeCoreConfiguration(c *config.EdgeCoreConfig) field.ErrorList {
	allErrs := field.ErrorList{}
	/*
		if !path.IsAbs(c.KubeConfig) {
			allErrs = append(allErrs, field.Invalid(field.NewPath("kubeConfig"), c.KubeConfig, "need abs path"))
		}
		if c.Port > 65535 {
			allErrs = append(allErrs, field.Invalid(field.NewPath("port"), c.Port, "port > 65535"))
		}
	*/
	return allErrs
}
