/*
Copyright 2022 The KubeEdge Authors.

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

package v1alpha2

import (
	"net"
	"net/url"
	"path"
	"path/filepath"
	"strconv"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kubeletconfigv1beta1 "k8s.io/kubelet/config/v1beta1"
	configv1beta1 "k8s.io/kubernetes/pkg/kubelet/apis/config/v1beta1"
	utilpointer "k8s.io/utils/pointer"

	"github.com/kubeedge/kubeedge/common/constants"
	metaconfig "github.com/kubeedge/kubeedge/pkg/apis/componentconfig/meta/v1alpha1"
	"github.com/kubeedge/kubeedge/pkg/util"
)

// NewDefaultEdgeCoreConfig returns a full EdgeCoreConfig object
func NewDefaultEdgeCoreConfig() *EdgeCoreConfig {
	hostnameOverride := util.GetHostname()
	localIP, _ := util.GetLocalIP(hostnameOverride)

	in := kubeletconfigv1beta1.KubeletConfiguration{}
	in.ContentType = "application/json"
	in.ImageGCLowThresholdPercent = utilpointer.Int32Ptr(constants.DefaultImageGCLowThreshold)
	in.ImageGCHighThresholdPercent = utilpointer.Int32Ptr(constants.DefaultImageGCHighThreshold)
	in.ConfigMapAndSecretChangeDetectionStrategy = kubeletconfigv1beta1.GetChangeDetectionStrategy
	in.FailSwapOn = utilpointer.BoolPtr(false)
	in.EnableServer = utilpointer.BoolPtr(false)
	in.Address = constants.ServerAddress
	in.ReadOnlyPort = constants.ServerPort
	in.ClusterDomain = constants.DefaultClusterDomain
	in.NodeStatusMaxImages = utilpointer.Int32Ptr(0)
	configv1beta1.SetDefaults_KubeletConfiguration(&in)

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
				Enable:                true,
				TailoredKubeletConfig: &in,
				TailoredKubeletFlag: TailoredKubeletFlag{
					KubeConfig:       constants.DefaultKubeletConfig,
					HostnameOverride: hostnameOverride,
					NodeIP:           "",
					ContainerRuntimeOptions: ContainerRuntimeOptions{
						ContainerRuntime:          constants.DefaultRuntimeType,
						DockerEndpoint:            constants.DefaultDockerAddress,
						PodSandboxImage:           constants.DefaultPodSandboxImage,
						ImagePullProgressDeadline: metav1.Duration{Duration: constants.DefaultImagePullProgressDeadline},
						CNIConfDir:                constants.DefaultCNIConfDir,
						CNIBinDir:                 constants.DefaultCNIBinDir,
						CNICacheDir:               constants.DefaultCNICacheDir,
						NetworkPluginMTU:          constants.DefaultNetworkPluginMTU,
					},
					RootDirectory:           "/var/lib/kubelet",
					MasterServiceNamespace:  metav1.NamespaceDefault,
					RemoteRuntimeEndpoint:   constants.DefaultRemoteRuntimeEndpoint,
					RemoteImageEndpoint:     constants.DefaultRemoteImageEndpoint,
					MaxPerPodContainerCount: 1,
					MinimumGCAge:            metav1.Duration{Duration: 0},
					NonMasqueradeCIDR:       "10.0.0.0/8",
					NodeLabels:              make(map[string]string),
					RegisterNode:            true,
					RegisterSchedulable:     true,
					SeccompProfileRoot:      filepath.Join("/var/lib/kubelet", "seccomp"),
				},
				CustomInterfaceName:   "",
				RegisterNodeNamespace: constants.DefaultRegisterNodeNamespace,
			},
			EdgeHub: &EdgeHub{
				Enable:            true,
				Heartbeat:         15,
				MessageQPS:        constants.DefaultQPS,
				MessageBurst:      constants.DefaultBurst,
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
				MqttSubClientID:      "",
				MqttPubClientID:      "",
				MqttUsername:         "",
				MqttPassword:         "",
				MqttMode:             MqttModeExternal,
				TLS: &EventBusTLS{
					Enable:                false,
					TLSMqttCAFile:         constants.DefaultMqttCAFile,
					TLSMqttCertFile:       constants.DefaultMqttCertFile,
					TLSMqttPrivateKeyFile: constants.DefaultMqttKeyFile,
				},
			},
			MetaManager: &MetaManager{
				Enable:             true,
				ContextSendGroup:   metaconfig.GroupNameHub,
				ContextSendModule:  metaconfig.ModuleNameEdgeHub,
				RemoteQueryTimeout: constants.DefaultRemoteQueryTimeout,
				MetaServer: &MetaServer{
					Enable:            false,
					Server:            constants.DefaultMetaServerAddr,
					TLSCaFile:         constants.DefaultCAFile,
					TLSCertFile:       constants.DefaultCertFile,
					TLSPrivateKeyFile: constants.DefaultKeyFile,
				},
			},
			ServiceBus: &ServiceBus{
				Enable:  false,
				Server:  "127.0.0.1",
				Port:    9060,
				Timeout: 60,
			},
			DeviceTwin: &DeviceTwin{
				Enable: true,
			},
			DBTest: &DBTest{
				Enable: false,
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
	hostnameOverride := util.GetHostname()
	localIP, _ := util.GetLocalIP(hostnameOverride)

	in := kubeletconfigv1beta1.KubeletConfiguration{}
	in.ContentType = "application/json"
	in.ImageGCLowThresholdPercent = utilpointer.Int32Ptr(constants.DefaultImageGCLowThreshold)
	in.ImageGCHighThresholdPercent = utilpointer.Int32Ptr(constants.DefaultImageGCHighThreshold)
	in.ConfigMapAndSecretChangeDetectionStrategy = kubeletconfigv1beta1.GetChangeDetectionStrategy
	in.FailSwapOn = utilpointer.BoolPtr(false)
	in.EnableServer = utilpointer.BoolPtr(false)
	in.Address = constants.ServerAddress
	in.ReadOnlyPort = constants.ServerPort
	in.ClusterDomain = constants.DefaultClusterDomain
	in.NodeStatusMaxImages = utilpointer.Int32Ptr(0)
	configv1beta1.SetDefaults_KubeletConfiguration(&in)

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
				Enable:                true,
				TailoredKubeletConfig: &in,
				TailoredKubeletFlag: TailoredKubeletFlag{
					KubeConfig:       constants.DefaultKubeletConfig,
					HostnameOverride: hostnameOverride,
					NodeIP:           "",
					ContainerRuntimeOptions: ContainerRuntimeOptions{
						ContainerRuntime:          constants.DefaultRuntimeType,
						DockerEndpoint:            constants.DefaultDockerAddress,
						PodSandboxImage:           constants.DefaultPodSandboxImage,
						ImagePullProgressDeadline: metav1.Duration{Duration: constants.DefaultImagePullProgressDeadline},
						CNIConfDir:                constants.DefaultCNIConfDir,
						CNIBinDir:                 constants.DefaultCNIBinDir,
						CNICacheDir:               constants.DefaultCNICacheDir,
						NetworkPluginMTU:          constants.DefaultNetworkPluginMTU,
					},
					RootDirectory:           "/var/lib/kubelet",
					MasterServiceNamespace:  metav1.NamespaceDefault,
					RemoteRuntimeEndpoint:   constants.DefaultRemoteRuntimeEndpoint,
					RemoteImageEndpoint:     constants.DefaultRemoteImageEndpoint,
					MaxPerPodContainerCount: 1,
					MinimumGCAge:            metav1.Duration{Duration: 0},
					NonMasqueradeCIDR:       "10.0.0.0/8",
					NodeLabels:              make(map[string]string),
					RegisterNode:            true,
					RegisterSchedulable:     true,
					SeccompProfileRoot:      filepath.Join("/var/lib/kubelet", "seccomp"),
				},
				CustomInterfaceName:   "",
				RegisterNodeNamespace: constants.DefaultRegisterNodeNamespace,
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
				MqttSubClientID:    "",
				MqttPubClientID:    "",
				MqttUsername:       "",
				MqttPassword:       "",
				MqttMode:           MqttModeExternal,
			},
		},
	}
}
