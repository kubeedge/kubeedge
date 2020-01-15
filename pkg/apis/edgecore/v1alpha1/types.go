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
	CGroupDriverCGroupFS = "cgroupfs"
	CGroupDriverSystemd  = "systemd"
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

// EdgeCoreConfig indicates the edgecore config which read from edgecore config file
type EdgeCoreConfig struct {
	metav1.TypeMeta
	// DataBase indicates database info
	// +Required
	DataBase *DataBase `json:"database,omitempty"`
	// Modules indicates cloudcore modules config
	// +Required
	Modules *Modules `json:"modules,omitempty"`
}

// DataBase indicates the datebase info
type DataBase struct {
	// DriverName indicates database driver name
	// default sqlite3
	DriverName string `json:"driverName,omitempty"`
	// AliasName indicates alias name
	// default default
	AliasName string `json:"aliasName,omitempty"`
	// DataSource indicates the data source path
	// default /var/lib/kubeedge/edge.db
	// Note: Can not use "omitempty" option,  It will affect the output of the default configuration file
	DataSource string `json:"dataSource,omitempty"`
}

// Modules indicates the modules which edgecore will be used
type Modules struct {
	// Edged indicates edged module config
	// +Required
	Edged *Edged `json:"edged,omitempty"`
	// EdgeHub indicates edgehub module config
	// +Required
	EdgeHub *EdgeHub `json:"edgehub,omitempty"`
	// EventBus indicates eventbus config for edgecore
	// +Required
	EventBus *EventBus `json:"eventbus,omitempty"`
	// MetaManager indicates meta module config
	// +Required
	MetaManager *MetaManager `json:"metamanager,omitempty"`
	// ServiceBus indicates module config
	ServiceBus *ServiceBus `json:"servicebus,omitempty"`
	// DeviceTwin indicates module config
	DeviceTwin *DeviceTwin `json:"devicetwin,omitempty"`
	// DBTest indicates module config
	DBTest *DBTest `json:"dbtest,omitempty"`
	// Mesh indicates mesh module config
	// +Required
	EdgeMesh *EdgeMesh `json:"edgemesh,omitempty"`
}

// Edged indicates the config fo edged module
// edged is lighted-kubelet
type Edged struct {
	// Enable indicates whether edged is enabled, if set to false (for debugging etc.), skip checking other edged configs.
	// default true
	Enable bool `json:"enable,omitempty"`
	// NodeStatusUpdateFrequency indicates node status update frequency (second)
	// default 10
	NodeStatusUpdateFrequency int32 `json:"nodeStatusUpdateFrequency,omitempty"`
	// RuntimeType indicates cri runtime ,support: docker, remote
	// default docker
	RuntimeType string `json:"runtimeType,omitempty"`
	// DockerAddress indicates docker server address
	// default unix:///var/run/docker.sock
	DockerAddress string `json:"dockerAddress,omitempty"`
	// RemoteRuntimeEndpoint indicates remote runtime endpoint
	// default unix:///var/run/dockershim.sock
	RemoteRuntimeEndpoint string `json:"remoteRuntimeEndpoint,omitempty"`
	// RemoteImageEndpoint indicates remote image endpoint
	// default unix:///var/run/dockershim.sock
	RemoteImageEndpoint string `json:"remoteImageEndpoint,omitempty"`
	// NodeIP indicates current node ip
	// default get local host ip
	NodeIP string `json:"nodeIP"`
	// ClusterDNS indicates cluster dns
	// Note: Can not use "omitempty" option,  It will affect the output of the default configuration file
	// +Required
	ClusterDNS string `json:"clusterDNS"`
	// ClusterDomain indicates cluster domain
	// Note: Can not use "omitempty" option,  It will affect the output of the default configuration file
	ClusterDomain string `json:"clusterDomain"`
	// EdgedMemoryCapacity indicates memory capacity (byte)
	// default 7852396000
	EdgedMemoryCapacity int64 `json:"edgedMemoryCapacity,omitempty"`
	// PodSandboxImage is the image whose network/ipc namespaces containers in each pod will use.
	// +Required
	// kubeedge/pause:3.1 for x86 arch
	// kubeedge/pause-arm:3.1 for arm arch
	// kubeedge/pause-arm64 for arm64 arch
	// default kubeedge/pause:3.1
	PodSandboxImage string `json:"podSandboxImage,omitempty"`
	// ImagePullProgressDeadline indicates image pull progress dead line (second)
	// default 60
	ImagePullProgressDeadline int32 `json:"imagePullProgressDeadline,omitempty"`
	// RuntimeRequestTimeout indicates runtime request timeout (second)
	// default 2
	RuntimeRequestTimeout int32 `json:"runtimeRequestTimeout,omitempty"`
	// HostnameOverride indicates hostname
	// default os.Hostname()
	HostnameOverride string `json:"hostnameOverride,omitempty"`
	//RegisterNodeNamespace indicates register node namespace
	// default default
	RegisterNodeNamespace string `json:"registerNodeNamespace,omitempty"`
	// InterfaceName indicates interface name
	// default eth0
	InterfaceName string `json:"interfaceName,omitempty"`
	// DevicePluginEnabled indicates enable device plugin
	// default false
	// Note: Can not use "omitempty" option,  It will affect the output of the default configuration file
	DevicePluginEnabled bool `json:"devicePluginEnabled"`
	// GPUPluginEnabled indicates enable gpu gplugin
	// default false,
	// Note: Can not use "omitempty" option,  It will affect the output of the default configuration file
	GPUPluginEnabled bool `json:"gpuPluginEnabled"`
	// ImageGCHighThreshold indicates image gc high threshold (percent)
	// default 80
	ImageGCHighThreshold int32 `json:"imageGCHighThreshold,omitempty"`
	// ImageGCLowThreshold indicates image gc low threshold (percent)
	// default 40
	ImageGCLowThreshold int32 `json:"imageGCLowThreshold,omitempty"`
	// MaximumDeadContainersPerPod indicates max num dead containers per pod
	// default 1
	MaximumDeadContainersPerPod int32 `json:"maximumDeadContainersPerPod,omitempty"`
	// CGroupDriver indicates container cgroup driver, support: cgroupfs,systemd
	// default cgroupfs
	// +Required
	CGroupDriver string `json:"cgroupDriver,omitempty"`
}

// EdgeHub indicates the edgehub module config
type EdgeHub struct {
	// Enable indicates whether edgehub is enabled, if set to false (for debugging etc.), skip checking other edgehub configs.
	// default true
	Enable bool `json:"enable,omitempty"`
	// Heartbeat indicates heart beat (second)
	// default 15
	Heartbeat int32 `json:"heartbeat,omitempty"`
	// ProjectID indicates project id
	// default e632aba927ea4ac2b575ec1603d56f10
	ProjectID string `json:"projectID,omitempty"`
	// TLSCAFile set ca file path
	// default /etc/kubeedge/ca/rootCA.crt
	TLSCAFile string `json:"tlsCaFile,omitempty"`
	// TLSCertFile indicates the file containing x509 Certificate for HTTPS
	// default /etc/kubeedge/certs/edge.crt
	TLSCertFile string `json:"tlsCertFile,omitempty"`
	// TLSPrivateKeyFile indicates the file containing x509 private key matching tlsCertFile
	// default /etc/kubeedge/certs/edge.key
	TLSPrivateKeyFile string `json:"tlsPrivateKeyFile,omitempty"`
	// Quic indicates quic config for edgehub module
	// Optional if websocket is configured
	Quic *EdgeHubQUIC `json:"quic,omitempty"`
	// WebSocket indicates websocket config for edgehub module
	// Optional if quic  is configured
	WebSocket *EdgeHubWebSocket `json:"websocket,omitempty"`
}

// EdgeHubQUIC indicates the quic client config
type EdgeHubQUIC struct {
	// Enable indicates whether enable this protocol
	// default true
	Enable bool `json:"enable,omitempty"`
	// HandshakeTimeout indicates hand shake timeout (second)
	// default 30
	HandshakeTimeout int32 `json:"handshakeTimeout,omitempty"`
	// ReadDeadline indicates read dead line (second)
	// default 15
	ReadDeadline int32 `json:"readDeadline,omitempty"`
	// Server indicates quic server addres (ip:port)
	// +Required
	Server string `json:"server,omitempty"`
	// WriteDeadline indicates write dead line (second)
	// default 15
	WriteDeadline int32 `json:"writeDeadline,omitempty"`
}

// EdgeHubWebSocket indicates the websocket client config
type EdgeHubWebSocket struct {
	// Enable indicates whether enable this protocol
	// default true
	Enable bool `json:"enable,omitempty"`
	// HandshakeTimeout indicates handshake timeout (second)
	// default  30
	HandshakeTimeout int32 `json:"handshakeTimeout,omitempty"`
	// ReadDeadline indicates read dead line (second)
	// default 15
	ReadDeadline int32 `json:"readDeadline,omitempty"`
	// Server indicates websocket server address (ip:port)
	// +Required
	Server string `json:"server,omitempty"`
	// WriteDeadline indicates write dead line (second)
	// default 15
	WriteDeadline int32 `json:"writeDeadline,omitempty"`
}

// EventBus indicates the event bus module config
type EventBus struct {
	// Enable indicates whether eventbus is enabled, if set to false (for debugging etc.), skip checking other eventbus configs.
	// default true
	Enable bool `json:"enable,omitempty"`
	// MqttQOS indicates mqtt qos
	// 0: QOSAtMostOnce, 1: QOSAtLeastOnce, 2: QOSExactlyOnce
	// default 0
	// Note: Can not use "omitempty" option,  It will affect the output of the default configuration file
	MqttQOS uint8 `json:"mqttQOS"`
	// MqttRetain indicates whether server will store the message and can be delivered to future subscribers
	// if this flag set true, sever will store the message and can be delivered to future subscribers
	// default false
	// Note: Can not use "omitempty" option,  It will affect the output of the default configuration file
	MqttRetain bool `json:"mqttRetain"`
	// MqttSessionQueueSize indicates the size of how many sessions will be handled.
	// default 100
	MqttSessionQueueSize int32 `json:"mqttSessionQueueSize,omitempty"`
	// MqttServerInternal indicates internal mqtt broker url
	// default tcp://127.0.0.1:1884
	MqttServerInternal string `json:"mqttServerInternal,omitempty"`
	// MqttServerExternal indicates external mqtt broker url
	// default tcp://127.0.0.1:1883
	MqttServerExternal string `json:"mqttServerExternal,omitempty"`
	// MqttMode indicates which broker type will be choose
	// 0: internal mqtt broker enable only. 1: internal and external mqtt broker enable. 2: external mqtt broker enable only
	// +Required
	// default: 0
	MqttMode MqttMode `json:"mqttMode,omitempty"`
}

// MetaManager indicates the metamanager module config
type MetaManager struct {
	// Enable indicates whether metamanager is enabled, if set to false (for debugging etc.), skip checking other metamanager configs.
	// default true
	Enable bool `json:"enable,omitempty"`
	// ContextSendGroup indicates send group
	ContextSendGroup metaconfig.GroupName `json:"contextSendGroup,omitempty"`
	// ContextSendModule indicates send module
	ContextSendModule metaconfig.ModuleName `json:"contextSendModule,omitempty"`
	// PodStatusSyncInterval indicates pod status sync
	PodStatusSyncInterval int32 `json:"podStatusSyncInterval,omitempty"`
}

// ServiceBus indicates the servicebus module config
type ServiceBus struct {
	// Enable indicates whether servicebus is enabled, if set to false (for debugging etc.), skip checking other servicebus configs.
	// default true
	Enable bool `json:"enable,omitempty"`
}

// DeviceTwin indicates the servicebus module config
type DeviceTwin struct {
	// Enable indicates whether devicetwin is enabled, if set to false (for debugging etc.), skip checking other devicetwin configs.
	// default true
	Enable bool `json:"enable,omitempty"`
}

// DBTest indicates the DBTest module config
type DBTest struct {
	// Enable indicates whether dbtest is enabled, if set to false (for debugging etc.), skip checking other dbtest configs.
	// default false
	Enable bool `json:"enable"`
}

// EdgeMesh indicates the edgemesh module config
type EdgeMesh struct {
	// Enable indicates whether edgemesh is enabled, if set to false (for debugging etc.), skip checking other edgemesh configs.
	// default true
	Enable bool `json:"enable,omitempty"`
	// lbStrategy indicates loadbalance stragety name
	LBStrategy string `json:"lbStrategy,omitempty"`
}
