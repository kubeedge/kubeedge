package validation

import (
	"path"

	"k8s.io/apimachinery/pkg/util/validation/field"

	"github.com/kubeedge/kubeedge/pkg/cloudcore/apis/config"
	commonconfig "github.com/kubeedge/kubeedge/pkg/common/apis/config"
)

func ValidateCloudCoreConfiguration(c *config.CloudCoreConfig) field.ErrorList {
	allErrs := field.ErrorList{}
	allErrs = append(allErrs, ValidateKubeConfiguration(c.Kube)...)
	allErrs = append(allErrs, ValidateEdgeControllerConfiguration(c.EdgeController)...)
	allErrs = append(allErrs, ValidateDeviceControllerConfiguration(c.DeviceController)...)
	allErrs = append(allErrs, ValidateCloudHubConfiguration(c.Cloudhub)...)
	allErrs = append(allErrs, ValidateModulesuration(c.Modules)...)
	return allErrs
}

func ValidateEdgeControllerConfiguration(e *config.EdgeControllerConfig) field.ErrorList {
	allErrs := field.ErrorList{}
	return allErrs
}

func ValidateDeviceControllerConfiguration(d *config.DeviceControllerConfig) field.ErrorList {
	allErrs := field.ErrorList{}
	return allErrs
}

func ValidateCloudHubConfiguration(c *config.CloudHubConfig) field.ErrorList {
	allErrs := field.ErrorList{}

	if c.WebsocketPort > 65535 {
		allErrs = append(allErrs, field.Invalid(field.NewPath("port"), c.WebsocketPort, "websocket port > 65535"))
	}

	if c.QuicPort > 65535 {
		allErrs = append(allErrs, field.Invalid(field.NewPath("port"), c.QuicPort, "quic port > 65535"))
	}

	return allErrs
}

func ValidateKubeConfiguration(k *config.KubeConfig) field.ErrorList {
	allErrs := field.ErrorList{}

	if !path.IsAbs(k.Kubeconfig) {
		allErrs = append(allErrs, field.Invalid(field.NewPath("kubeconfig"), k.Kubeconfig, "kubeconfig need abs path"))
	}
	return allErrs
}

func ValidateModulesuration(m *commonconfig.Modules) field.ErrorList {
	allErrs := field.ErrorList{}
	return allErrs
}

func ValidateAdmissionControllerConfiguration(a *config.AdmissionControllerConfig) field.ErrorList {
	allErrs := field.ErrorList{}
	return allErrs
}
