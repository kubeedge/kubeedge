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
		DataBase: DataBase{
			DriverName: DataBaseDriverName,
			AliasName:  DataBaseAliasName,
			DataSource: DataBaseDataSource,
		},
		Modules: EdgeCoreModules{
			Edged: Edged{
				Enable:                      true,
				NodeStatusUpdateFrequency:   10,
				DockerAddress:               "unix:///var/run/docker.sock",
				RuntimeType:                 "docker",
				NodeIP:                      "",
				ClusterDNS:                  "",
				ClusterDomain:               "",
				EdgedMemoryCapacity:         7852396000,
				RemoteRuntimeEndpoint:       "unix:///var/run/dockershim.sock",
				RemoteImageEndpoint:         "unix:///var/run/dockershim.sock",
				PodSandboxImage:             "kubeedge/pause:3.1",
				ImagePullProgressDeadline:   60,
				RuntimeRequestTimeout:       2,
				HostnameOverride:            "edge-node",
				RegisterNodeNamespace:       "default",
				InterfaceName:               "eth0",
				DevicePluginEnabled:         false,
				GPUPluginEnabled:            false,
				ImageGCHighThreshold:        80,
				ImageGCLowThreshold:         40,
				MaximumDeadContainersPerPod: 1,
				CGroupDriver:                "cgroupfs",
			},
			EdgeHub: EdgeHub{
				Enable:            true,
				Heartbeat:         15,
				ProjectID:         "e632aba927ea4ac2b575ec1603d56f10",
				TLSCAFile:         constants.DefaultCAFile,
				TLSCertFile:       constants.DefaultCertFile,
				TLSPrivateKeyFile: constants.DefaultKeyFile,
				Quic: EdgeHubQuic{
					Enable:           false,
					HandshakeTimeout: 30,
					ReadDeadline:     15,
					Server:           "127.0.0.1:10001",
					WriteDeadline:    15,
				},
				WebSocket: EdgeHubWebSocket{
					Enable:           true,
					HandshakeTimeout: 30,
					ReadDeadline:     15,
					Server:           "127.0.0.1:10000",
					WriteDeadline:    15,
				},
			},
			EventBus: EventBus{
				Enable:               true,
				MqttQOS:              0,
				MqttRetain:           false,
				MqttSessionQueueSize: 100,
				MqttServerExternal:   "tcp://127.0.0.1:1883",
				MqttServerInternal:   "tcp://127.0.0.1:1884",
				MqttMode:             MqttModeExternal,
			},
			MetaManager: MetaManager{
				Enable:                true,
				ContextSendGroup:      metaconfig.GroupNameHub,
				ContextSendModule:     metaconfig.ModuleNameEdgeHub,
				PodStatusSyncInterval: 60,
			},
			ServiceBus: ServiceBus{
				Enable: true,
			},
			DeviceTwin: DeviceTwin{
				Enable: true,
			},
			DBTest: DBTest{
				Enable: true,
			},
			EdgeMesh: EdgeMesh{
				Enable:     true,
				LBStrategy: LoadBalanceStrategNameRoundRobin,
			},
		},
	}
}
