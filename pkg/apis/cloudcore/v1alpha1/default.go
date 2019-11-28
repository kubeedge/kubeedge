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

package v1alpha1

import (
	"path"

	"github.com/kubeedge/kubeedge/common/constants"
	metaconfig "github.com/kubeedge/kubeedge/pkg/apis/meta/v1alpha1"
)

// NewDefaultCloudCoreConfig return a default CloudCoreConfig object
func NewDefaultCloudCoreConfig() *CloudCoreConfig {
	return &CloudCoreConfig{
		TypeMeta: metaconfig.TypeMeta{
			Kind:       Kind,
			APIVersion: path.Join(GroupName, APIVersion),
		},
		Kube:             newDefaultKubeConfig(),
		EdgeController:   newDefaultEdgeControllerConfig(),
		DeviceController: newDefaultDeviceControllerConfig(),
		Cloudhub:         newDefaultCloudHubConfig(),
		Modules:          newDefaultModules(),
	}
}

// newDefaultEdgeControllerConfig return a default EdgeControllerConfig object
func newDefaultEdgeControllerConfig() EdgeControllerConfig {
	return EdgeControllerConfig{
		NodeUpdateFrequency: 10,
		ControllerContext: ControllerContext{
			SendModule:     metaconfig.ModuleNameCloudHub,
			ReceiveModule:  metaconfig.ModuleNameEdgeController,
			ResponseModule: metaconfig.ModuleNameCloudHub,
		},
	}
}

// newDefaultDeviceControllerConfig return a default DeviceControllerConfig object
func newDefaultDeviceControllerConfig() DeviceControllerConfig {
	return DeviceControllerConfig{
		ControllerContext: ControllerContext{
			SendModule:     metaconfig.ModuleNameCloudHub,
			ReceiveModule:  metaconfig.ModuleNameDeviceController,
			ResponseModule: metaconfig.ModuleNameCloudHub,
		},
	}
}

// newDefaultCloudHubConfig return a default CloudHubConfig object
func newDefaultCloudHubConfig() CloudHubConfig {
	return CloudHubConfig{
		EnableWebsocket:    true,
		WebsocketPort:      10000,
		EnableQuic:         false,
		QuicPort:           10001,
		MaxIncomingStreams: 10000,
		EnableUnixSocket:   true,
		UnixSocketAddress:  "unix:///var/lib/kubeedge/kubeedge.sock",
		Address:            "0.0.0.0",
		TLSCAFile:          path.Join(constants.DefaultCADir, "rootCA.crt"),
		TLSCertFile:        path.Join(constants.DefaultCertDir, "edge.crt"),
		TLSPrivateKeyFile:  path.Join(constants.DefaultCertDir, "edge.key"),
		KeepaliveInterval:  30,
		WriteTimeout:       30,
		NodeLimit:          10,
	}
}

// newDefaultKubeConfig return a default KubeConfig object
func newDefaultKubeConfig() KubeConfig {
	return KubeConfig{
		Master:     "",
		KubeConfig: "/root/.kube/config",
	}
}

// newDefaultModules return a default Modules object
func newDefaultModules() metaconfig.Modules {
	return metaconfig.Modules{
		Enabled: []metaconfig.ModuleName{
			metaconfig.ModuleNameDeviceController,
			metaconfig.ModuleNameEdgeController,
			metaconfig.ModuleNameCloudHub,
		},
	}
}
