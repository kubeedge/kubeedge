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
	metaconfig "github.com/kubeedge/kubeedge/pkg/apis/meta/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	LoadBalanceStrategNameRoundRobin string = "RoundRobin"
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
	DataBaseDataSource = "/var/lib/kubeedge/edgecore.db"
)

type ProtocolName string
type MqttMode int

type EdgeCoreConfig struct {
	metav1.TypeMeta
	// DataBase set database info
	// +Required
	DataBase DataBase `json:"database,omitempty"`
	// Modules set cloudcore modules config
	// +Required
	Modules EdgeCoreModules `json:"modules,omitempty"`
}

type DataBase struct {
	// DriverName set database driver name
	// default sqlite3
	DriverName string `json:"driverName,omitempty"`
	// AliasName set alias name
	// default default
	AliasName string `json:"aliasName,omitempty"`
	// DataSource set the data source path
	// default /var/lib/kubeedge/edge.db
	// Note: Can not use "omitempty" option,  It will affect the output of the default configuration file
	DataSource string `json:"dataSource,omitempty"`
}

type EdgeCoreModules struct {
	// Edged set edged module config
	// +Required
	Edged Edged `json:"edged,omitempty"`
	// EdgeHub set edgehub module config
	// +Required
	EdgeHub EdgeHub `json:"edgehub,omitempty"`
	// EventBus set eventbus config for edgecore
	// +Required
	EventBus EventBus `json:"eventbus,omitempty"`
	// MetaManager set meta module config
	// +Required
	MetaManager MetaManager `json:"metamanager,omitempty"`
	// ServiceBus set module config
	// +Required
	ServiceBus ServiceBus `json:"servicebus,omitempty"`
	// DeviceTwin set module config
	// +Required
	DeviceTwin DeviceTwin `json:"servicebus,omitempty"`
	// DBTest set module config
	// +Required
	DBTest DBTest `json:"dbtest,omitempty"`
	// Mesh set mesh module config
	// +Required
	EdgeMesh EdgeMesh `json:"edgemesh,omitempty"`
}

type Edged struct {
	// Enable set whether use this module, if false , need check other config
	// default true
	Enable bool `json:"enable,omitempty"`
	// NodeStatusUpdateFrequency set node status update frequency (second)
	// default 10
	NodeStatusUpdateFrequency int32 `json:"nodeStatusUpdateFrequency,omitempty"`
	// DockerAddress set docker server address
	// default unix:///var/run/docker.sock
	DockerAddress string `json:"dockerAddress,omitempty"`
	// RuntimeType set cri runtime ,support: docker, remote
	// default docker
	RuntimeType string `json:"runtimeType,omitempty"`
	// NodeIP set current node ip
	// Note: Can not use "omitempty" option,  It will affect the output of the default configuration file
	NodeIP string `json:"nodeIP"`
	// ClusterDNS set cluster dns
	// Note: Can not use "omitempty" option,  It will affect the output of the default configuration file
	ClusterDNS string `json:"clusterDNS"`
	// ClusterDomain set cluster domain
	// Note: Can not use "omitempty" option,  It will affect the output of the default configuration file
	ClusterDomain string `json:"clusterDomain"`
	// EdgedMemoryCapacity set memory capacity (byte)
	// default 7852396000
	EdgedMemoryCapacity int64 `json:"edgedMemoryCapacity,omitempty"`
	// RemoteRuntimeEndpoint set remote runtime endpoint
	// default unix:///var/run/dockershim.sock
	RemoteRuntimeEndpoint string `json:"remoteRuntimeEndpoint,omitempty"`
	// RemoteImageEndpoint set remote image endpoint
	// default unix:///var/run/dockershim.sock
	RemoteImageEndpoint string `json:"remoteImageEndpoint,omitempty"`
	// PodSandboxImage is the image whose network/ipc namespaces containers in each pod will use.
	// kubeedge/pause:3.1 for x86 arch
	// kubeedge/pause-arm:3.1 for arm arch
	// kubeedge/pause-arm64 for arm64 arch
	// default kubeedge/pause:3.1
	PodSandboxImage string `json:"podSandboxImage,omitempty"`
	// ImagePullProgressDeadline set image pull progress dead line (second)
	// default 60
	ImagePullProgressDeadline int32 `json:"imagePullProgressDeadline,omitempty"`
	// RuntimeRequestTimeout set runtime request timeout (second)
	// default 2
	RuntimeRequestTimeout int32 `json:"runtimeRequestTimeout,omitempty"`
	// HostnameOverride set hostname
	// default edge-node
	HostnameOverride string `json:"hostnameOverride,omitempty"`
	//RegisterNodeNamespace set register node namespace
	// default default
	RegisterNodeNamespace string `json:"registerNodeNamespace,omitempty"`
	// InterfaceName set interface name
	// default eth0
	InterfaceName string `json:"interfaceName,omitempty"`
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
	// CGroupDriver set container cgroup driver, support: cgroupfs,systemd
	// default cgroupfs
	CGroupDriver string `json:"cgroupDriver,omitempty"`
}

type EdgeHub struct {
	// Enable set whether use this module, if false , need check other config
	// default true
	Enable bool `json:"enable,omitempty"`
	// Heartbeat set heart beat (second)
	// default 15
	Heartbeat int32 `json:"heartbeat,omitempty"`
	// ProjectID set project id
	// default e632aba927ea4ac2b575ec1603d56f10
	ProjectID string `json:"projectID,omitempty"`
	// ProjectId set project id
	// default e632aba927ea4ac2b575ec1603d56f10
	// TLSCAFile set ca file path
	// default /etc/kubeedge/ca/rootCA.crt
	TLSCAFile string `json:"tlsCaFile,omitempty"`
	// TLSCertFile is the file containing x509 Certificate for HTTPS
	// default /etc/kubeedge/certs/edge.crt
	TLSCertFile string `json:"tlsCertFile,omitempty"`
	// TLSPrivateKeyFile is the file containing x509 private key matching tlsCertFile
	// default /etc/kubeedge/certs/edge.key
	TLSPrivateKeyFile string `json:"tlsPrivateKeyFile,omitempty"`
	// Quic set quic config for edgehub module
	Quic EdgeHubQuic `json:"quic,omitempty"`
	// WebSocket set websocket config for edgehub module
	WebSocket EdgeHubWebSocket `json:"webSocket,omitempty"`
}
type EdgeHubQuic struct {
	// Enable enable this protocol
	// default true
	// Note: Can not use "omitempty" option,  It will affect the output of the default configuration file
	Enable bool `json:"enable"`
	// HandshakeTimeout set hand shake timeout (second)
	// default 30
	HandshakeTimeout int32 `json:"handshakeTimeout,omitempty"`
	// ReadDeadline set read dead line (second)
	// default 15
	ReadDeadline int32 `json:"readDeadline,omitempty"`
	// Server set quic server addres (ip:port)
	Server string `json:"server,omitempty"`
	// WriteDeadline set write dead linke (second)
	// default 15
	WriteDeadline int32 `json:"writeDeadline,omitempty"`
}
type EdgeHubWebSocket struct {
	// Enable enable this protocol
	// default true
	// Note: Can not use "omitempty" option,  It will affect the output of the default configuration file
	Enable bool `json:"enable"`
	// HandshakeTimeout set handshake timeout (second)
	// default  30
	HandshakeTimeout int32 `json:"handshakeTimeout,omitempty"`
	// ReadDeadline set read dead line (second)
	// default 15
	ReadDeadline int32 `json:"readDeadline,omitempty"`
	// Server set websocket server address (ip:port)
	Server string `json:"server,omitempty"`
	// WriteDeadline set write dead line (second)
	// default 15
	WriteDeadline int32 `json:"writeDeadline,omitempty"`
}

type EventBus struct {
	// Enable set whether use this module, if false , need check other config
	// default true
	Enable bool `json:"enable,omitempty"`
	// MqttQOS set mqtt qos
	// 0: QOSAtMostOnce, 1: QOSAtLeastOnce, 2: QOSExactlyOnce
	// default 0
	// Note: Can not use "omitempty" option,  It will affect the output of the default configuration file
	MqttQOS uint8 `json:"mqttQOS"`
	// MqttRetain set whether server will store the message and can be delivered to future subscribers
	// if this flag set true, sever will store the message and can be delivered to future subscribers
	// default false
	// Note: Can not use "omitempty" option,  It will affect the output of the default configuration file
	MqttRetain bool `json:"mqttRetain"`
	// MqttSessionQueueSize set size of how many sessions will be handled.
	// default 100
	MqttSessionQueueSize int32 `json:"mqttSessionQueueSize,omitempty"`
	// MqttServerInternal set internal mqtt broker url
	// default tcp://127.0.0.1:1884
	MqttServerInternal string `json:"mqttServerInternal,omitempty"`
	// MqttServerExternal set external mqtt broker url
	// default tcp://127.0.0.1:1883
	MqttServerExternal string `json:"mqttServerExternal,omitempty"`
	// MqttMode set which broker type will be choose
	// 0: internal mqtt broker enable only. 1: internal and external mqtt broker enable. 2: external mqtt broker enable only
	// default: 0
	MqttMode MqttMode `json:"mqttMode,omitempty"`
}

type MetaManager struct {
	// Enable set whether use this module, if false , need check other config
	// default true
	Enable bool `json:"enable,omitempty"`
	// ContextSendGroup set send group
	ContextSendGroup metaconfig.GroupName `json:"contextSendGroup,omitempty"`
	// ContextSendModule set send module
	ContextSendModule metaconfig.ModuleName `json:"contextSendModule,omitempty"`
	// PodStatusSyncInterval set pod status sync
	PodStatusSyncInterval int32 `json:"podStatusSyncInterval,omitempty"`
}

type ServiceBus struct {
	// Enable set whether use this module, if false , need check other config
	// default true
	Enable bool `json:"enable,omitempty"`
}
type DeviceTwin struct {
	// Enable set whether use this module, if false , need check other config
	// default true
	Enable bool `json:"enable,omitempty"`
}
type DBTest struct {
	// Enable set whether use this module, if false , need check other config
	// default true
	Enable bool `json:"enable,omitempty"`
}

type EdgeMesh struct {
	// Enable set whether use this module, if false , need check other config
	// default true
	Enable bool `json:"enable,omitempty"`
	// lbStrategy set loadbalance stragety name
	// +Required
	LBStrategy string `json:"lbStrategy,omitempty"`
}
