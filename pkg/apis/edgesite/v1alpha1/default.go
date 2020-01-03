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

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	cloudcoreconfig "github.com/kubeedge/kubeedge/pkg/apis/cloudcore/v1alpha1"
	edgecoreconfig "github.com/kubeedge/kubeedge/pkg/apis/edgecore/v1alpha1"
	metaconfig "github.com/kubeedge/kubeedge/pkg/apis/meta/v1alpha1"
)

// NewDefaultEdgeSideConfig return a default EdgeSideConfig object
func NewDefaultEdgeSideConfig() *EdgeSideConfig {
	return &EdgeSideConfig{
		TypeMeta: metav1.TypeMeta{
			Kind:       Kind,
			APIVersion: path.Join(GroupName, APIVersion),
		},
		Mqtt:              newDefaultMqttConfig(),
		Kube:              newDefaultKubeConfig(),
		ControllerContext: newDefaultControllerContext(),
		Edged:             newDefaultEdgedConfig(),
		Modules:           newDefaultModules(),
		MetaManager:       newDefaultMetamanager(),
		DataBase:          newDefaultDataBase(),
	}
}

// newDefaultEdgedConfig return a default Edged object
func newDefaultEdgedConfig() edgecoreconfig.Edged {
	return edgecoreconfig.Edged{
		HostnameOverride:            "edge-node",
		InterfaceName:               "eth0",
		EdgedMemoryCapacity:         7852396000,
		NodeStatusUpdateFrequency:   10,
		DevicePluginEnabled:         false,
		GPUPluginEnabled:            false,
		ImageGCHighThreshold:        80,
		ImageGCLowThreshold:         40,
		MaximumDeadContainersPerPod: 1,
		DockerAddress:               "unix:///var/run/docker.sock",
		RuntimeType:                 "docker",
		RemoteRuntimeEndpoint:       "unix:///var/run/dockershim.sock",
		RemoteImageEndpoint:         "unix:///var/run/dockershim.sock",
		RuntimeRequestTimeout:       2,
		PodSandboxImage:             "kubeedge/pause:3.1",
		ImagePullProgressDeadline:   60,
		CGroupDriver:                "cgroupfs",
		NodeIP:                      "127.0.0.1",
		ClusterDNS:                  "8.8.8.8",
		ClusterDomain:               "",
	}
}

// newDefaultKubeConfig return a default KubeConfig object
func newDefaultKubeConfig() cloudcoreconfig.KubeConfig {
	return cloudcoreconfig.KubeConfig{
		Master:     "",
		KubeConfig: "/root/.kube/config",
	}
}

// newDefaultMqttConfig return a default MqttConfig object
func newDefaultMqttConfig() edgecoreconfig.MqttConfig {
	return edgecoreconfig.MqttConfig{
		Server:           "tcp://127.0.0.1:1883",
		InternalServer:   "tcp://127.0.0.1:1884",
		Mode:             edgecoreconfig.MqttModeExternal,
		QOS:              0,
		Retain:           false,
		SessionQueueSize: 100,
	}
}

// newDefaultControllerContext return a default EdgeControllerContext object
func newDefaultControllerContext() cloudcoreconfig.EdgeControllerContext {
	return cloudcoreconfig.EdgeControllerContext{
		SendModule:     "metaManager",
		ReceiveModule:  "edgecontroller",
		ResponseModule: "metaManager",
	}
}

// newDefaultModules return a default Modules object
func newDefaultModules() metaconfig.Modules {
	return metaconfig.Modules{
		Enabled: []metaconfig.ModuleName{
			metaconfig.ModuleNameEdgeController,
			metaconfig.ModuleNameMetaManager,
			metaconfig.ModuleNameEdged,
			metaconfig.ModuleNameDBTest,
		},
	}
}

// newDefaultMetamanager return a default MetaManager object
func newDefaultMetamanager() edgecoreconfig.MetaManager {
	return edgecoreconfig.MetaManager{
		ContextSendGroup:  metaconfig.GroupNameEdgeController,
		ContextSendModule: metaconfig.ModuleNameEdgeController,
		EdgeSite:          true,
	}
}

// newDefaultDataBase return a default DataBase object
func newDefaultDataBase() edgecoreconfig.DataBase {
	return edgecoreconfig.DataBase{
		DriverName: DataBaseDriverName,
		AliasName:  DataBaseAliasName,
		DataSource: DataBaseDataSource,
	}
}
