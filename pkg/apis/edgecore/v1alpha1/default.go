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

	"github.com/kubeedge/kubeedge/common/constants"
	metaconfig "github.com/kubeedge/kubeedge/pkg/apis/meta/v1alpha1"
)

// NewDefaultEdgeCoreConfig return a default EdgeCoreConfig object
func NewDefaultEdgeCoreConfig() *EdgeCoreConfig {
	return &EdgeCoreConfig{
		TypeMeta: metav1.TypeMeta{
			Kind:       Kind,
			APIVersion: path.Join(GroupName, APIVersion),
		},
		Mqtt:        newDefaultMqttConfig(),
		EdgeHub:     newDefaultEdgeHubConfig(),
		Edged:       newDefaultEdgedConfig(),
		Mesh:        newDefaultMeshConfig(),
		Modules:     newDefaultModules(),
		MetaManager: newDefaultMetamanager(),
		DataBase:    newDefaultDataBase(),
	}
}

// newDefaultMqttConfig return a default MqttConfig object
func newDefaultMqttConfig() MqttConfig {
	return MqttConfig{
		Server:           "tcp://127.0.0.1:1883",
		InternalServer:   "tcp://127.0.0.1:1884",
		Mode:             MqttModeExternal,
		QOS:              0,
		Retain:           false,
		SessionQueueSize: 100,
	}
}

// newDefaultEdgeHubConfig return a default EdgeHubConfig object
func newDefaultEdgeHubConfig() EdgeHubConfig {
	return EdgeHubConfig{
		WebSocket:         newDefaultWebSocketConfig(),
		Quic:              newDefaultQuicConfig(),
		TLSCaFile:         constants.DefaultCAFile,
		TLSCertFile:       constants.DefaultCertFile,
		TLSPrivateKeyFile: constants.DefaultKeyFile,
		Protocol:          ProtocolNameWebSocket,
		Heartbeat:         15,
	}
}

// newDefaultWebSocketConfig return a default WebSocketConfig object
func newDefaultWebSocketConfig() WebSocketConfig {
	return WebSocketConfig{
		Server:           "127.0.0.1:10000",
		HandshakeTimeout: 30,
		WriteDeadline:    15,
		ReadDeadline:     15,
	}
}

// newDefaultQuicConfig return a default QuicConfig object
func newDefaultQuicConfig() QuicConfig {
	return QuicConfig{
		Server:           "127.0.0.1:10001",
		HandshakeTimeout: 30,
		WriteDeadline:    15,
		ReadDeadline:     15,
	}
}

// newDefaultEdgedConfig return a default EdgedConfig object
func newDefaultEdgedConfig() EdgedConfig {
	return EdgedConfig{
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
		CgroupDriver:                "cgroupfs",
		NodeIP:                      "127.0.0.1",
		ClusterDNS:                  "8.8.8.8",
		ClusterDomain:               "",
	}
}

// newDefaultMeshConfig return a default MeshConfig object
func newDefaultMeshConfig() MeshConfig {
	return MeshConfig{
		Loadbalance: newDefaultLoadbalanceConfig(),
	}
}

// newDefaultLoadbalanceConfig return a default LoadbalanceConfig object
func newDefaultLoadbalanceConfig() LoadbalanceConfig {
	return LoadbalanceConfig{
		StrategyName: LoadBalanceStrategNameRoundRobin,
	}
}

// newDefaultModules return a default Modules object
func newDefaultModules() metaconfig.Modules {
	return metaconfig.Modules{
		Enabled: []metaconfig.ModuleName{
			metaconfig.ModuleNameEventBus,
			metaconfig.ModuleNameServiceBus,
			metaconfig.ModuleNameEdgeHub,
			metaconfig.ModuleNameMetaManager,
			metaconfig.ModuleNameEdged,
			metaconfig.ModuleNameTwin,
			metaconfig.ModuleNameDBTest,
			metaconfig.ModuleNameEdgeMesh,
		},
	}
}

// newDefaultMetamanager return a default Metamanager object
func newDefaultMetamanager() MetaManager {
	return MetaManager{
		ContextSendGroup:  metaconfig.GroupNameHub,
		ContextSendModule: metaconfig.ModuleNameEdgeHub,
		EdgeSite:          false,
	}
}

// newDefaultDataBase return a default DataBase object
func newDefaultDataBase() DataBase {
	return DataBase{
		DriverName: DataBaseDriverName,
		AliasName:  DataBaseAliasName,
		DataSource: DataBaseDataSource,
	}
}
