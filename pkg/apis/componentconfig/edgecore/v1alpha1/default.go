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
	"net"
	"net/url"
	"os"
	"path"
	"strconv"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/kubeedge/kubeedge/common/constants"
	metaconfig "github.com/kubeedge/kubeedge/pkg/apis/componentconfig/meta/v1alpha1"
	"github.com/kubeedge/kubeedge/pkg/util"
)

// NewDefaultEdgeCoreConfig returns a full EdgeCoreConfig object
func NewDefaultEdgeCoreConfig() *EdgeCoreConfig {
	hostnameOverride, err := os.Hostname()
	if err != nil {
		hostnameOverride = constants.DefaultHostnameOverride
	}
	localIP, _ := util.GetLocalIP(hostnameOverride)

	return &EdgeCoreConfig{
		TypeMeta: metav1.TypeMeta{
			Kind:       Kind,
			APIVersion: path.Join(GroupName, APIVersion),
		},
		DataBase: &DataBase{
			DriverName: DataBaseDriverName,
			AliasName:  DataBaseAliasName,
			DataSource: DataBaseDataSource,
		},
		Modules: &Modules{
			Edged: &Edged{
				Enable:                      true,
				NodeStatusUpdateFrequency:   constants.DefaultNodeStatusUpdateFrequency,
				RuntimeType:                 constants.DefaultRuntimeType,
				DockerAddress:               constants.DefaultDockerAddress,
				RemoteRuntimeEndpoint:       constants.DefaultRemoteRuntimeEndpoint,
				RemoteImageEndpoint:         constants.DefaultRemoteImageEndpoint,
				NodeIP:                      localIP,
				ClusterDNS:                  "",
				ClusterDomain:               "",
				ConcurrentConsumers:         constants.DefaultConcurrentConsumers,
				EdgedMemoryCapacity:         constants.DefaultEdgedMemoryCapacity,
				PodSandboxImage:             util.GetPodSandboxImage(),
				ImagePullProgressDeadline:   constants.DefaultImagePullProgressDeadline,
				RuntimeRequestTimeout:       constants.DefaultRuntimeRequestTimeout,
				HostnameOverride:            hostnameOverride,
				RegisterNodeNamespace:       constants.DefaultRegisterNodeNamespace,
				RegisterNode:                true,
				DevicePluginEnabled:         false,
				GPUPluginEnabled:            false,
				ImageGCHighThreshold:        constants.DefaultImageGCHighThreshold,
				ImageGCLowThreshold:         constants.DefaultImageGCLowThreshold,
				MaximumDeadContainersPerPod: constants.DefaultMaximumDeadContainersPerPod,
				CGroupDriver:                CGroupDriverCGroupFS,
				CgroupsPerQOS:               true,
				CgroupRoot:                  constants.DefaultCgroupRoot,
				NetworkPluginName:           "",
				CNIConfDir:                  constants.DefaultCNIConfDir,
				CNIBinDir:                   constants.DefaultCNIBinDir,
				CNICacheDir:                 constants.DefaultCNICacheDir,
				NetworkPluginMTU:            constants.DefaultNetworkPluginMTU,
				VolumeStatsAggPeriod:        constants.DefaultVolumeStatsAggPeriod,
				EnableMetrics:               true,
			},
			EdgeHub: &EdgeHub{
				Enable:            true,
				Heartbeat:         15,
				ProjectID:         "e632aba927ea4ac2b575ec1603d56f10",
				TLSCAFile:         constants.DefaultCAFile,
				TLSCertFile:       constants.DefaultCertFile,
				TLSPrivateKeyFile: constants.DefaultKeyFile,
				Quic: &EdgeHubQUIC{
					Enable:           false,
					HandshakeTimeout: 30,
					ReadDeadline:     15,
					Server:           net.JoinHostPort(localIP, "10001"),
					WriteDeadline:    15,
				},
				WebSocket: &EdgeHubWebSocket{
					Enable:           true,
					HandshakeTimeout: 30,
					ReadDeadline:     15,
					Server:           net.JoinHostPort(localIP, "10000"),
					WriteDeadline:    15,
				},
				HTTPServer: (&url.URL{
					Scheme: "https",
					Host:   net.JoinHostPort(localIP, "10002"),
				}).String(),
				Token:              "",
				RotateCertificates: true,
			},
			EventBus: &EventBus{
				Enable:               true,
				MqttQOS:              0,
				MqttRetain:           false,
				MqttSessionQueueSize: 100,
				MqttServerExternal:   "tcp://127.0.0.1:1883",
				MqttServerInternal:   "tcp://127.0.0.1:1884",
				MqttMode:             MqttModeExternal,
				TLS: &EventBusTLS{
					Enable:                false,
					TLSMqttCAFile:         constants.DefaultMqttCAFile,
					TLSMqttCertFile:       constants.DefaultMqttCertFile,
					TLSMqttPrivateKeyFile: constants.DefaultMqttKeyFile,
				},
			},
			MetaManager: &MetaManager{
				Enable:                true,
				ContextSendGroup:      metaconfig.GroupNameHub,
				ContextSendModule:     metaconfig.ModuleNameEdgeHub,
				PodStatusSyncInterval: constants.DefaultPodStatusSyncInterval,
				RemoteQueryTimeout:    constants.DefaultRemoteQueryTimeout,
			},
			ServiceBus: &ServiceBus{
				Enable: false,
			},
			DeviceTwin: &DeviceTwin{
				Enable: true,
			},
			DBTest: &DBTest{
				Enable: false,
			},
			EdgeMesh: &EdgeMesh{
				Enable:          true,
				LBStrategy:      EdgeMeshDefaultLoadBalanceStrategy,
				ListenInterface: EdgeMeshDefaultInterface,
				ListenPort:      EdgeMeshDefaultListenPort,
				SubNet:          EdgeMeshDefaultSubNet,
			},
			EdgeStream: &EdgeStream{
				Enable:                  false,
				TLSTunnelCAFile:         constants.DefaultCAFile,
				TLSTunnelCertFile:       constants.DefaultCertFile,
				TLSTunnelPrivateKeyFile: constants.DefaultKeyFile,
				HandshakeTimeout:        30,
				ReadDeadline:            15,
				TunnelServer:            net.JoinHostPort("127.0.0.1", strconv.Itoa(constants.DefaultTunnelPort)),
				WriteDeadline:           15,
			},
		},
	}
}

// NewMinEdgeCoreConfig returns a common EdgeCoreConfig object
func NewMinEdgeCoreConfig() *EdgeCoreConfig {
	hostnameOverride, err := os.Hostname()
	if err != nil {
		hostnameOverride = constants.DefaultHostnameOverride
	}
	localIP, _ := util.GetLocalIP(hostnameOverride)
	return &EdgeCoreConfig{
		TypeMeta: metav1.TypeMeta{
			Kind:       Kind,
			APIVersion: path.Join(GroupName, APIVersion),
		},
		DataBase: &DataBase{
			DataSource: DataBaseDataSource,
		},
		Modules: &Modules{
			Edged: &Edged{
				RuntimeType:           constants.DefaultRuntimeType,
				RemoteRuntimeEndpoint: constants.DefaultRemoteRuntimeEndpoint,
				RemoteImageEndpoint:   constants.DefaultRemoteImageEndpoint,
				DockerAddress:         constants.DefaultDockerAddress,
				NodeIP:                localIP,
				ClusterDNS:            "",
				ClusterDomain:         "",
				PodSandboxImage:       util.GetPodSandboxImage(),
				HostnameOverride:      hostnameOverride,
				DevicePluginEnabled:   false,
				GPUPluginEnabled:      false,
				CGroupDriver:          CGroupDriverCGroupFS,
				CgroupsPerQOS:         true,
				CgroupRoot:            constants.DefaultCgroupRoot,
			},
			EdgeHub: &EdgeHub{
				Heartbeat:         15,
				TLSCAFile:         constants.DefaultCAFile,
				TLSCertFile:       constants.DefaultCertFile,
				TLSPrivateKeyFile: constants.DefaultKeyFile,
				WebSocket: &EdgeHubWebSocket{
					Enable:           true,
					HandshakeTimeout: 30,
					ReadDeadline:     15,
					Server:           net.JoinHostPort(localIP, "10000"),
					WriteDeadline:    15,
				},
				HTTPServer: (&url.URL{
					Scheme: "https",
					Host:   net.JoinHostPort(localIP, "10002"),
				}).String(),
				Token: "",
			},
			EventBus: &EventBus{
				MqttQOS:            0,
				MqttRetain:         false,
				MqttServerExternal: "tcp://127.0.0.1:1883",
				MqttServerInternal: "tcp://127.0.0.1:1884",
				MqttMode:           MqttModeExternal,
			},
		},
	}
}
