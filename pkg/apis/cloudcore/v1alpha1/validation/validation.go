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
	metaconfig "github.com/kubeedge/kubeedge/pkg/apis/meta/v1alpha1"
	utilvalidation "github.com/kubeedge/kubeedge/pkg/util/validation"
)

// ValidateCloudCoreConfiguration validates `c` and returns an errorList if it is invalid
func ValidateCloudCoreConfiguration(c *cloudconfig.CloudCoreConfig) field.ErrorList {
	allErrs := field.ErrorList{}
	allErrs = append(allErrs, ValidateKubeConfiguration(c.Kube)...)
	allErrs = append(allErrs, ValidateEdgeControllerConfiguration(c.EdgeController)...)
	allErrs = append(allErrs, ValidateDeviceControllerConfiguration(c.DeviceController)...)
	allErrs = append(allErrs, ValidateCloudHubConfiguration(c.Cloudhub)...)
	allErrs = append(allErrs, ValidateModulesuration(c.Modules)...)
	return allErrs
}

// ValidateEdgeControllerConfiguration validates `e` and returns an errorList if it is invalid
func ValidateEdgeControllerConfiguration(e cloudconfig.EdgeControllerConfig) field.ErrorList {
	allErrs := field.ErrorList{}
	switch {
	case e.NodeUpdateFrequency <= 0:
		allErrs = append(allErrs, field.Invalid(field.NewPath("NodeUpdateFrequency"), e.NodeUpdateFrequency, "NodeUpdateFrequency need > 0"))
		fallthrough
	default:
		allErrs = append(allErrs, ValidateControllerContext(e.ControllerContext)...)
	}
	return allErrs
}

// ValidateDeviceControllerConfiguration validates `d` and returns an errorList if it is invalid
func ValidateDeviceControllerConfiguration(d cloudconfig.DeviceControllerConfig) field.ErrorList {
	allErrs := field.ErrorList{}
	allErrs = append(allErrs, ValidateControllerContext(d.ControllerContext)...)
	return allErrs
}

// ValidateCloudHubConfiguration validates `c` and returns an errorList if it is invalid
func ValidateCloudHubConfiguration(c cloudconfig.CloudHubConfig) field.ErrorList {
	allErrs := field.ErrorList{}

	allErrs = append(allErrs, ValidateCloudHubWebSocket(c.WebSocket)...)
	allErrs = append(allErrs, ValidateCloudHubQuic(c.Quic)...)
	allErrs = append(allErrs, ValidateCloudHubUnixSocket(c.UnixSocket)...)

	switch {
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
	return allErrs
}

// ValidateCloudHubWebSocket validates `ws` and returns an errorList if it is invalid
func ValidateCloudHubWebSocket(ws cloudconfig.CloudHubWebSocket) field.ErrorList {
	allErrs := field.ErrorList{}
	validWPort := utilvalidation.IsValidPortNum(int(ws.WebsocketPort))
	validAddress := utilvalidation.IsValidIP(ws.Address)
	switch {
	case len(validWPort) > 0:
		for _, m := range validWPort {
			allErrs = append(allErrs, field.Invalid(field.NewPath("port"), ws.WebsocketPort, m))
		}
		fallthrough
	case len(validAddress) > 0:
		for _, m := range validAddress {
			allErrs = append(allErrs, field.Invalid(field.NewPath("Address"), ws.Address, m))
		}
		fallthrough
	default:
	}
	return allErrs
}

// ValidateCloudHubQuic validates `q` and returns an errorList if it is invalid
func ValidateCloudHubQuic(q cloudconfig.CloudHubQuic) field.ErrorList {
	allErrs := field.ErrorList{}
	validQPort := utilvalidation.IsValidPortNum(int(q.QuicPort))
	validAddress := utilvalidation.IsValidIP(q.Address)
	switch {
	case len(validQPort) > 0:
		for _, m := range validQPort {
			allErrs = append(allErrs, field.Invalid(field.NewPath("port"), q.QuicPort, m))
		}
		fallthrough
	case len(validAddress) > 0:
		for _, m := range validAddress {
			allErrs = append(allErrs, field.Invalid(field.NewPath("Address"), q.Address, m))
		}
		fallthrough
	default:
	}
	return allErrs
}

// ValidateCloudHubUnixSocket validates `us` and returns an errorList if it is invalid
func ValidateCloudHubUnixSocket(us cloudconfig.CloudHubUnixSocket) field.ErrorList {
	allErrs := field.ErrorList{}

	if !strings.HasPrefix(strings.ToLower(us.UnixSocketAddress), "unix://") {
		allErrs = append(allErrs, field.Invalid(field.NewPath("unixSocketAddress"),
			us.UnixSocketAddress, "unixSocketAddress must has prefix unix://"))
	}
	s := strings.SplitN(us.UnixSocketAddress, "://", 2)
	if len(s) > 1 && !utilvalidation.FileIsExist(path.Dir(s[1])) {
		allErrs = append(allErrs, field.Invalid(field.NewPath("unixSocketAddress"),
			us.UnixSocketAddress,
			fmt.Sprintf("unixSocketAddress %v dir %v not exist , need create it",
				us.UnixSocketAddress, path.Dir(s[1]))))
	}
	return allErrs
}

// ValidateKubeConfiguration validates `k` and returns an errorList if it is invalid
func ValidateKubeConfiguration(k cloudconfig.KubeConfig) field.ErrorList {
	allErrs := field.ErrorList{}
	switch {
	case !path.IsAbs(k.KubeConfig):
		allErrs = append(allErrs, field.Invalid(field.NewPath("kubeconfig"), k.KubeConfig, "kubeconfig need abs path"))
		fallthrough
	case !utilvalidation.FileIsExist(k.KubeConfig):
		allErrs = append(allErrs, field.Invalid(field.NewPath("kubeconfig"), k.KubeConfig, "kubeconfig not exist"))
		fallthrough
	default:

	}
	return allErrs
}

// ValidateModulesuration validates `m` and returns an errorList if it is invalid
func ValidateModulesuration(m metaconfig.Modules) field.ErrorList {
	allErrs := field.ErrorList{}
	switch {
	case len(m.Enabled) == 0:
		allErrs = append(allErrs, field.Invalid(field.NewPath("Enabled"), m.Enabled, "Enabled Modules should be set"))
		fallthrough
	default:

	}
	return allErrs
}

// ValidateControllerContext validates `c` and returns an errorList if it is invalid
func ValidateControllerContext(c cloudconfig.ControllerContext) field.ErrorList {
	allErrs := field.ErrorList{}
	return allErrs
}
