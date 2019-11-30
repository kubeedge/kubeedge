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

import metaconfig "github.com/kubeedge/kubeedge/pkg/apis/meta/v1alpha1"

const (
	LoadBalanceStrategNameRoundRobin LoadBalanceStrategName = "RoundRobin"
)

const (
	MqttModeInternal MqttMode = 0
	MqttModeBoth     MqttMode = 1
	MqttModeExternal MqttMode = 2
)

const (
	ProtocolNameWebSocket ProtocolName = "websocket"
	ProtocolNameQuic      ProtocolName = "quic"
)

const (
	// DataBaseDriverName is sqlite3
	DataBaseDriverName = "sqlite3"
	// DataBaseAliasName is default
	DataBaseAliasName = "default"
	// DataBaseDataSource is edge.db
	DataBaseDataSource = "/var/lib/kubeedge/edge.db"
)

type LoadBalanceStrategName string
type ProtocolName string
type MqttMode int

type EdgeCoreConfig struct {
	metaconfig.TypeMeta
	// Mqtt set mqtt config for edgecore
	// +Required
	Mqtt MqttConfig `json:"mqtt,omitempty"`
	// EdgeHub set edgehub module config
	// +Required
	EdgeHub EdgeHubConfig `json:"edgehub,omitempty"`
	// Edged set edged module config
	// +Required
	Edged EdgedConfig `json:"edged,omitempty"`
	// Mesh set mesh module config
	// +Required
	Mesh MeshConfig `json:"mesh,omitempty"`
	// Modules set which modules are enabled
	// +Required
	Modules metaconfig.Modules `json:"modules,omitempty"`
	// MetaManager set meta module config
	// +Required
	MetaManager MetaManager `json:"metaManager,omitempty"`
	// DataBase set DataBase config
	// +Required
	DataBase DataBase `json:"database,omitempty"`
}

type MqttConfig struct {
	// Server set external mqtt broker url
	// default tcp://127.0.0.1:1883
	Server string `json:"server,omitempty"`
	// InternalServer set internal mqtt broker url
	// default tcp://127.0.0.1:1884
	InternalServer string `json:"internalServer,omitempty"`
	// Mode set which broker type will be choose
	// 0: internal mqtt broker enable only. 1: internal and external mqtt broker enable. 2: external mqtt broker enable only
	// default: 0
	Mode MqttMode `json:"mode,omitempty"`
	// QOS set mqtt qos
	// 0: QOSAtMostOnce, 1: QOSAtLeastOnce, 2: QOSExactlyOnce
	// default 0
	// Note: Can not use "omitempty" option,  It will affect the output of the default configuration file
	QOS uint8 `json:"qos"`
	// Retain set whether server will store the message and can be delivered to future subscribers
	// if this flag set true, sever will store the message and can be delivered to future subscribers
	// default false
	// Note: Can not use "omitempty" option,  It will affect the output of the default configuration file
	Retain bool `json:"retain"`
	// SessionQueueSize set size of how many sessions will be handled.
	// default 100
	SessionQueueSize int32 `json:"sessionQueueSize,omitempty"`
}

type EdgeHubConfig struct {
	// WebSocket set websocket config for edgehub module
	WebSocket WebSocketConfig `json:"webSocket,omitempty"`
	// Quic set quic config for edgehub module
	Quic QuicConfig `json:"quic,omitempty"`
	// Protocol set which protocol will be use, now support:websocket, quic
	// default: websocket
	Protocol ProtocolName `json:"protocol,omitempty"`
	// Heartbeat set heart beat (second)
	// default 15
	Heartbeat int32 `json:"heartbeat,omitempty"`
	// ProjectId set project id
	// default e632aba927ea4ac2b575ec1603d56f10
	// TLSCaFile set ca file path
	// default /etc/kubeedge/ca/rootCA.crt
	TLSCaFile string `json:"tlsCaFile,omitempty"`
	// TLSCertFile is the file containing x509 Certificate for HTTPS
	// default /etc/kubeedge/certs/edge.crt
	TLSCertFile string `json:"tlsCertFile,omitempty"`
	// TLSPrivateKeyFile is the file containing x509 private key matching tlsCertFile
	// default /etc/kubeedge/certs/edge.key
	TLSPrivateKeyFile string `json:"tlsPrivateKeyFile,omitempty"`
}

type WebSocketConfig struct {
	// Server set websocket server address (ip:port)
	Server string `json:"server,omitempty"`
	// HandshakeTimeout set handshake timeout (second)
	// default  30
	HandshakeTimeout int32 `json:"handshakeTimeout,omitempty"`
	// WriteDeadline set write dead line (second)
	// default 15
	WriteDeadline int32 `json:"writeDeadline,omitempty"`
	// ReadDeadline set read dead line (second)
	// default 15
	ReadDeadline int32 `json:"readDeadline,omitempty"`
}

type QuicConfig struct {
	// Server set quic server addres (ip:port)
	Server string `json:"server,omitempty"`
	// HandshakeTimeout set hand shake timeout (second)
	// default 30
	HandshakeTimeout int32 `json:"handshakeTimeout,omitempty"`
	// WriteDeadline set write dead linke (second)
	// default 15
	WriteDeadline int32 `json:"writeDeadline,omitempty"`
	// ReadDeadline set read dead line (second)
	// default 15
	ReadDeadline int32 `json:"readDeadline,omitempty"`
}

type EdgedConfig struct {
	//RegisterNodeNamespace set register node namespace
	// default default
	// HostnameOverride set hostname
	// default edge-node
	HostnameOverride string `json:"hostnameOverride,omitempty"`
	// InterfaceName set interface name
	// default eth0
	InterfaceName string `json:"interfaceName,omitempty"`
	// EdgedMemoryCapacity set memory capacity (byte)
	// default 7852396000
	EdgedMemoryCapacity int64 `json:"edgedMemoryCapacity,omitempty"`
	// NodeStatusUpdateFrequency set node status update frequency (second)
	// default 10
	NodeStatusUpdateFrequency int32 `json:"nodeStatusUpdateFrequency,omitempty"`
	// DevicePluginEnabled set enable device plugin
	// default false
	// Note: Can not use "omitempty" option,  It will affect the output of the default configuration file
	DevicePluginEnabled bool `json:"devicePluginEnabled"`
	// GPUPluginEnabled set enable gpu gplugin
	// default false,
	// Note: Can not use "omitempty" option,  It will affect the output of the default configuration file
	GPUPluginEnabled bool `json:"gpuPluginEnabled"`
	// ImageGCHighThreshold set image gc high threshold (percent)
	// default 80
	ImageGCHighThreshold int32 `json:"imageGCHighThreshold,omitempty"`
	// ImageGCLowThreshold set image gc low threshold (percent)
	// default 40
	ImageGCLowThreshold int32 `json:"imageGCLowThreshold,omitempty"`
	// MaximumDeadContainersPerPod set max num dead containers per pod
	// default 1
	MaximumDeadContainersPerPod int32 `json:"maximumDeadContainersPerPod,omitempty"`
	// DockerAddress set docker server address
	// default unix:///var/run/docker.sock
	DockerAddress string `json:"dockerAddress,omitempty"`
	// RuntimeType set cri runtime ,support: docker, remote
	// default docker
	RuntimeType string `json:"runtimeType,omitempty"`
	// RemoteRuntimeEndpoint set remote runtime endpoint
	// default unix:///var/run/dockershim.sock
	RemoteRuntimeEndpoint string `json:"remoteRuntimeEndpoint,omitempty"`
	// RemoteImageEndpoint set remote image endpoint
	// default unix:///var/run/dockershim.sock
	RemoteImageEndpoint string `json:"remoteImageEndpoint,omitempty"`
	// RuntimeRequestTimeout set runtime request timeout (second)
	// default 2
	RuntimeRequestTimeout int32 `json:"runtimeRequestTimeout,omitempty"`
	// PodSandboxImage is the image whose network/ipc namespaces containers in each pod will use.
	// kubeedge/pause:3.1 for x86 arch
	// kubeedge/pause-arm:3.1 for arm arch
	// kubeedge/pause-arm64 for arm64 arch
	// default kubeedge/pause:3.1
	PodSandboxImage string `json:"podSandboxImage,omitempty"`
	// ImagePullProgressDeadline set image pull progress dead line (second)
	// default 60
	ImagePullProgressDeadline int32 `json:"imagePullProgressDeadline,omitempty"`
	// CgroupDriver set container cgroup driver, support: cgroupfs,systemd
	// default cgroupfs
	CgroupDriver string `json:"cgroupDriver,omitempty"`
	// NodeIP set current node ip
	// Note: Can not use "omitempty" option,  It will affect the output of the default configuration file
	NodeIP string `json:"nodeIP"`
	// ClusterDNS set cluster dns
	// Note: Can not use "omitempty" option,  It will affect the output of the default configuration file
	ClusterDNS string `json:"clusterDNS"`
	// ClusterDomain set cluster domain
	// Note: Can not use "omitempty" option,  It will affect the output of the default configuration file
	ClusterDomain string `json:"clusterDomain"`
}

type MeshConfig struct {
	// Loadbalance set loadbalance config
	// +Required
	Loadbalance LoadbalanceConfig `json:"loadbalance,omitempty"`
}

type LoadbalanceConfig struct {
	// StrategyName set loadbalance stragey
	// default RoundRobin
	StrategyName LoadBalanceStrategName `json:"strategyName,omitempty"`
}

type MetaManager struct {
	// ContextSendGroup set send group
	ContextSendGroup metaconfig.GroupName `json:"contextSendGroup,omitempty"`
	// ContextSendModule set send module
	ContextSendModule metaconfig.ModuleName `json:"contextSendModule,omitempty"`
	// EdgeSite standfor whether set edgesite
	// default false
	// Note: Can not use "omitempty" option,  It will affect the output of the default configuration file
	EdgeSite bool `json:"edgeSite"`
}

type DataBase struct {
	// DriverName set database driver name
	// default sqlite3
	DriverName string `json:"drivername,omitempty"`
	// AliasName set alias name
	// default default
	AliasName string `json:"aliasname,omitempty"`
	// DataSource set the data source path
	// default /var/lib/kubeedge/edge.db
	// Note: Can not use "omitempty" option,  It will affect the output of the default configuration file
	DataSource string `json:"datasource,omitempty"`
}
