/*
Copyright 2019 The KubeEdge Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

   http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package validation

import (
	"fmt"

	"k8s.io/apimachinery/pkg/util/validation/field"

	edgecoreconfig "github.com/kubeedge/kubeedge/pkg/apis/edgecore/v1alpha1"
)

// ValidateEdgeCoreConfiguration validates `c` and returns an errorList if it is invalid
func ValidateEdgeCoreConfiguration(c *edgecoreconfig.EdgeCoreConfig) field.ErrorList {
	allErrs := field.ErrorList{}
	allErrs = append(allErrs, ValidateMqttConfiguration(c.Mqtt)...)
	allErrs = append(allErrs, ValidateEdgeHubConfiguration(c.EdgeHub)...)
	allErrs = append(allErrs, ValidateEdgedConfiguration(c.Edged)...)
	allErrs = append(allErrs, ValidateMeshConfiguration(c.Mesh)...)
	return allErrs
}

// ValidateMqttConfiguration validates `m` and returns an errorList if it is invalid
func ValidateMqttConfiguration(m edgecoreconfig.MqttConfig) field.ErrorList {
	allErrs := field.ErrorList{}
	switch {
	case m.Mode > edgecoreconfig.MqttModeExternal || m.Mode < edgecoreconfig.MqttModeInternal:
		allErrs = append(allErrs, field.Invalid(field.NewPath("Mode"), m.Mode,
			fmt.Sprintf("Mode need in [%v,%v] range", edgecoreconfig.MqttModeInternal,
				edgecoreconfig.MqttModeExternal)))
		fallthrough
	default:

	}
	return allErrs
}

// ValidateEdgeHubConfiguration validates `h` and returns an errorList if it is invalid
func ValidateEdgeHubConfiguration(h edgecoreconfig.EdgeHubConfig) field.ErrorList {
	allErrs := field.ErrorList{}
	return allErrs
}

// ValidateEdgedConfiguration validates `e` and returns an errorList if it is invalid
func ValidateEdgedConfiguration(e edgecoreconfig.EdgedConfig) field.ErrorList {

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

// ValidateMeshConfiguration validates `m` and returns an errorList if it is invalid
func ValidateMeshConfiguration(m edgecoreconfig.MeshConfig) field.ErrorList {
	// TODO check meshconfig @kadisi
	allErrs := field.ErrorList{}
	return allErrs
}
