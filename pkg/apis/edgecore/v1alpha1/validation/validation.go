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
	"os"
	"path"
	"strings"

	"k8s.io/apimachinery/pkg/util/validation/field"

	edgecoreconfig "github.com/kubeedge/kubeedge/pkg/apis/edgecore/v1alpha1"
	utilvalidation "github.com/kubeedge/kubeedge/pkg/util/validation"
)

// ValidateEdgeCoreConfiguration validates `c` and returns an errorList if it is invalid
func ValidateEdgeCoreConfiguration(c *edgecoreconfig.EdgeCoreConfig) field.ErrorList {
	allErrs := field.ErrorList{}
	allErrs = append(allErrs, ValidateDataBase(c.DataBase)...)
	allErrs = append(allErrs, ValidateModuleEdged(c.Modules.Edged)...)
	allErrs = append(allErrs, ValidateModuleEdgeHub(c.Modules.EdgeHub)...)
	allErrs = append(allErrs, ValidateModuleEventBus(c.Modules.EventBus)...)
	allErrs = append(allErrs, ValidateModuleMetaManager(c.Modules.MetaManager)...)
	allErrs = append(allErrs, ValidateModuleServiceBus(c.Modules.ServiceBus)...)
	allErrs = append(allErrs, ValidateModuleDeviceTwin(c.Modules.DeviceTwin)...)
	allErrs = append(allErrs, ValidateModuleDBTest(c.Modules.DBTest)...)
	allErrs = append(allErrs, ValidateModuleEdgeMesh(c.Modules.EdgeMesh)...)
	return allErrs
}

// ValidateDataBase validates `db` and returns an errorList if it is invalid
func ValidateDataBase(db edgecoreconfig.DataBase) field.ErrorList {
	allErrs := field.ErrorList{}
	sourceDir := path.Dir(db.DataSource)
	if !utilvalidation.FileIsExist(sourceDir) {
		if err := os.MkdirAll(sourceDir, os.ModePerm); err != nil {
			allErrs = append(allErrs, field.Invalid(field.NewPath("DataSource"), db.DataSource,
				fmt.Sprintf("create DataSoure dir %v error ", sourceDir)))
		}
	}
	return allErrs
}

// ValidateModuleEdged validates `e` and returns an errorList if it is invalid
func ValidateModuleEdged(e edgecoreconfig.Edged) field.ErrorList {
	if !e.Enable {
		return field.ErrorList{}
	}
	allErrs := field.ErrorList{}
	if e.NodeIP == "" {
		allErrs = append(allErrs, field.Invalid(field.NewPath("NodeIp"), e.NodeIP,
			"Need sed NodeIP"))
	}
	switch e.CGroupDriver {
	case "cgroupfs", "systemd":
	default:
		allErrs = append(allErrs, field.Invalid(field.NewPath("CGroupDriver"), e.CGroupDriver,
			"CGroupDriver value error"))
	}
	return allErrs
}

// ValidateModuleEdgeHub validates `h` and returns an errorList if it is invalid
func ValidateModuleEdgeHub(h edgecoreconfig.EdgeHub) field.ErrorList {
	if !h.Enable {
		return field.ErrorList{}
	}
	allErrs := field.ErrorList{}
	quicIP := strings.SplitN(h.Quic.Server, ":", 1)[0]
	wsIP := strings.SplitN(h.WebSocket.Server, ":", 1)[0]
	validQAddress := utilvalidation.IsValidIP(quicIP)
	validWSAddress := utilvalidation.IsValidIP(wsIP)

	switch {
	case !utilvalidation.FileIsExist(h.TLSPrivateKeyFile):
		allErrs = append(allErrs, field.Invalid(field.NewPath("TLSPrivateKeyFile"),
			h.TLSPrivateKeyFile, "TLSPrivateKeyFile not exist"))
		fallthrough
	case !utilvalidation.FileIsExist(h.TLSCertFile):
		allErrs = append(allErrs, field.Invalid(field.NewPath("TLSCertFile"),
			h.TLSCertFile, "TLSCertFile not exist"))
		fallthrough
	case !utilvalidation.FileIsExist(h.TLSCAFile):
		allErrs = append(allErrs, field.Invalid(field.NewPath("TLSCAFile"),
			h.TLSCAFile, "TLSCAFile not exist"))
		fallthrough
	case len(validQAddress) > 0:
		for _, m := range validQAddress {
			allErrs = append(allErrs, field.Invalid(field.NewPath("quic.server"), h.Quic.Server, m))
		}
		fallthrough
	case len(validWSAddress) > 0:
		for _, m := range validWSAddress {
			allErrs = append(allErrs, field.Invalid(field.NewPath("websocket.server"), h.WebSocket.Server, m))
		}
		fallthrough
	case h.WebSocket.Enable == h.Quic.Enable:
		allErrs = append(allErrs, field.Invalid(field.NewPath("enable"),
			h.Quic.Enable, "websocket.enable and quic.enable cannot be true and false at the same time"))
		fallthrough
	default:
	}

	return allErrs
}

// ValidateModuleEventBus validates `m` and returns an errorList if it is invalid
func ValidateModuleEventBus(m edgecoreconfig.EventBus) field.ErrorList {
	if !m.Enable {
		return field.ErrorList{}
	}
	allErrs := field.ErrorList{}
	switch {
	case m.MqttMode > edgecoreconfig.MqttModeExternal || m.MqttMode < edgecoreconfig.MqttModeInternal:
		allErrs = append(allErrs, field.Invalid(field.NewPath("Mode"), m.MqttMode,
			fmt.Sprintf("Mode need in [%v,%v] range", edgecoreconfig.MqttModeInternal,
				edgecoreconfig.MqttModeExternal)))
		fallthrough
	default:

	}
	return allErrs
}

// ValidateModuleMetaManager validates `m` and returns an errorList if it is invalid
func ValidateModuleMetaManager(m edgecoreconfig.MetaManager) field.ErrorList {
	if !m.Enable {
		return field.ErrorList{}
	}
	allErrs := field.ErrorList{}
	return allErrs
}

// ValidateModuleServiceBus validates `s` and returns an errorList if it is invalid
func ValidateModuleServiceBus(s edgecoreconfig.ServiceBus) field.ErrorList {
	if !s.Enable {
		return field.ErrorList{}
	}
	allErrs := field.ErrorList{}
	return allErrs
}

// ValidateModuleDeviceTwin validates `d` and returns an errorList if it is invalid
func ValidateModuleDeviceTwin(d edgecoreconfig.DeviceTwin) field.ErrorList {
	if !d.Enable {
		return field.ErrorList{}
	}
	allErrs := field.ErrorList{}
	return allErrs
}

// ValidateModuleDBTest validates `d` and returns an errorList if it is invalid
func ValidateModuleDBTest(d edgecoreconfig.DBTest) field.ErrorList {
	if !d.Enable {
		return field.ErrorList{}
	}
	allErrs := field.ErrorList{}
	return allErrs
}

// ValidateModuleEdgeMesh validates `m` and returns an errorList if it is invalid
func ValidateModuleEdgeMesh(m edgecoreconfig.EdgeMesh) field.ErrorList {
	if !m.Enable {
		return field.ErrorList{}
	}
	// TODO check meshconfig @kadisi
	allErrs := field.ErrorList{}
	return allErrs
}
