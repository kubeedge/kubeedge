package validation

import (
	"fmt"

	"k8s.io/apimachinery/pkg/util/validation/field"

	"github.com/kubeedge/kubeedge/pkg/edgecore/apis/config"
)

func ValidateEdgeCoreConfiguration(c *config.EdgeCoreConfig) field.ErrorList {
	allErrs := field.ErrorList{}
	allErrs = append(allErrs, ValidateMqttConfiguration(c.Mqtt)...)
	allErrs = append(allErrs, ValidateEdgeHubConfiguration(c.EdgeHub)...)
	allErrs = append(allErrs, ValidateEdgedConfiguration(c.Edged)...)
	allErrs = append(allErrs, ValidateMeshConfiguration(c.Mesh)...)
	return allErrs
}

func ValidateMqttConfiguration(m *config.MqttConfig) field.ErrorList {
	allErrs := field.ErrorList{}
	if m.Mode > config.ExternalMqttMode || m.Mode < config.InternalMqttMode {
		allErrs = append(allErrs, field.Invalid(field.NewPath("Mode"), m.Mode,
			fmt.Sprintf("Mode need >= %v && <= %v", config.InternalMqttMode, config.ExternalMqttMode)))
	}
	return allErrs
}

func ValidateEdgeHubConfiguration(h *config.EdgeHubConfig) field.ErrorList {
	allErrs := field.ErrorList{}
	return allErrs
}

func ValidateEdgedConfiguration(e *config.EdgedConfig) field.ErrorList {

	allErrs := field.ErrorList{}
	if e.NodeIP == "" {
		allErrs = append(allErrs, field.Invalid(field.NewPath("NodeIp"), e.NodeIP,
			"Need sed NodeIP"))
	}
	switch e.CgroupDriver {
	case "cgroupfs", "systemd":
	default:
		allErrs = append(allErrs, field.Invalid(field.NewPath("CgroupDriver"), e.CgroupDriver,
			"CgroupDriver value error"))
	}
	return allErrs
}

func ValidateMeshConfiguration(m *config.MeshConfig) field.ErrorList {
	// TODO check meshconfig @kadisi
	allErrs := field.ErrorList{}
	return allErrs
}
