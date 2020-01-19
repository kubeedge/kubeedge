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
	"path"
	"strings"

	"k8s.io/apimachinery/pkg/util/validation/field"

	cloudconfig "github.com/kubeedge/kubeedge/pkg/apis/cloudcore/v1alpha1"
	utilvalidation "github.com/kubeedge/kubeedge/pkg/util/validation"
)

// ValidateCloudCoreConfiguration validates `c` and returns an errorList if it is invalid
func ValidateCloudCoreConfiguration(c *cloudconfig.CloudCoreConfig) field.ErrorList {
	allErrs := field.ErrorList{}
	allErrs = append(allErrs, ValidateKubeAPIConfig(*c.KubeAPIConfig)...)
	allErrs = append(allErrs, ValidateModuleCloudHub(*c.Modules.CloudHub)...)
	allErrs = append(allErrs, ValidateModuleEdgeController(*c.Modules.EdgeController)...)
	allErrs = append(allErrs, ValidateModuleDeviceController(*c.Modules.DeviceController)...)
	return allErrs
}

// ValidateModuleCloudHub validates `c` and returns an errorList if it is invalid
func ValidateModuleCloudHub(c cloudconfig.CloudHub) field.ErrorList {
	if !c.Enable {
		return field.ErrorList{}
	}

	allErrs := field.ErrorList{}
	validWPort := utilvalidation.IsValidPortNum(int(c.WebSocket.Port))
	validAddress := utilvalidation.IsValidIP(c.WebSocket.Address)
	validQPort := utilvalidation.IsValidPortNum(int(c.Quic.Port))
	validQAddress := utilvalidation.IsValidIP(c.Quic.Address)

	switch {
	case len(validWPort) > 0:
		for _, m := range validWPort {
			allErrs = append(allErrs, field.Invalid(field.NewPath("port"), c.WebSocket.Port, m))
		}
		fallthrough
	case len(validAddress) > 0:
		for _, m := range validAddress {
			allErrs = append(allErrs, field.Invalid(field.NewPath("Address"), c.WebSocket.Address, m))
		}
		fallthrough
	case len(validQPort) > 0:
		for _, m := range validQPort {
			allErrs = append(allErrs, field.Invalid(field.NewPath("port"), c.Quic.Port, m))
		}
		fallthrough
	case len(validQAddress) > 0:
		for _, m := range validQAddress {
			allErrs = append(allErrs, field.Invalid(field.NewPath("Address"), c.Quic.Address, m))
		}
		fallthrough
	case !utilvalidation.FileIsExist(c.TLSPrivateKeyFile):
		allErrs = append(allErrs, field.Invalid(field.NewPath("TLSPrivateKeyFile"), c.TLSPrivateKeyFile, "TLSPrivateKeyFile not exist"))
		fallthrough
	case !utilvalidation.FileIsExist(c.TLSCertFile):
		allErrs = append(allErrs, field.Invalid(field.NewPath("TLSCertFile"), c.TLSCertFile, "TLSCertFile not exist"))
		fallthrough
	case !utilvalidation.FileIsExist(c.TLSCAFile):
		allErrs = append(allErrs, field.Invalid(field.NewPath("TLSCAFile"), c.TLSCAFile, "TLSCAFile not exist"))
		fallthrough
	default:
	}
	if !strings.HasPrefix(strings.ToLower(c.UnixSocket.Address), "unix://") {
		allErrs = append(allErrs, field.Invalid(field.NewPath("address"),
			c.UnixSocket.Address, "unixSocketAddress must has prefix unix://"))
	}
	s := strings.SplitN(c.UnixSocket.Address, "://", 2)
	if len(s) > 1 && !utilvalidation.FileIsExist(path.Dir(s[1])) {
		allErrs = append(allErrs, field.Invalid(field.NewPath("address"),
			c.UnixSocket.Address, fmt.Sprintf("unixSocketAddress %v dir %v not exist , need create it",
				c.UnixSocket.Address, path.Dir(s[1]))))
	}
	return allErrs
}

// ValidateModuleEdgeController validates `e` and returns an errorList if it is invalid
func ValidateModuleEdgeController(e cloudconfig.EdgeController) field.ErrorList {
	if !e.Enable {
		return field.ErrorList{}
	}
	allErrs := field.ErrorList{}
	switch {
	case e.NodeUpdateFrequency <= 0:
		allErrs = append(allErrs, field.Invalid(field.NewPath("NodeUpdateFrequency"), e.NodeUpdateFrequency, "NodeUpdateFrequency need > 0"))
		fallthrough
	default:
	}
	return allErrs
}

// ValidateModuleDeviceController validates `d` and returns an errorList if it is invalid
func ValidateModuleDeviceController(d cloudconfig.DeviceController) field.ErrorList {
	if !d.Enable {
		return field.ErrorList{}
	}

	allErrs := field.ErrorList{}
	return allErrs
}

// ValidateKubeAPIConfig validates `k` and returns an errorList if it is invalid
func ValidateKubeAPIConfig(k cloudconfig.KubeAPIConfig) field.ErrorList {
	allErrs := field.ErrorList{}
	switch {
	case k.KubeConfig != "" && !path.IsAbs(k.KubeConfig):
		allErrs = append(allErrs, field.Invalid(field.NewPath("kubeconfig"), k.KubeConfig, "kubeconfig need abs path"))
		fallthrough
	case k.KubeConfig != "" && !utilvalidation.FileIsExist(k.KubeConfig):
		allErrs = append(allErrs, field.Invalid(field.NewPath("kubeconfig"), k.KubeConfig, "kubeconfig not exist"))
		fallthrough
	default:

	}
	return allErrs
}
