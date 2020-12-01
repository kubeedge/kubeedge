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
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	metaconfig "github.com/kubeedge/kubeedge/pkg/apis/componentconfig/meta/v1alpha1"
)

const (
	EdgeMeshDefaultLoadBalanceStrategy = "RoundRobin"
	EdgeMeshDefaultInterface           = "docker0"
	EdgeMeshDefaultSubNet              = "9.251.0.0/16"
	EdgeMeshDefaultListenPort          = 40001
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

// EdgeCoreConfig indicates the EdgeCore config which read from EdgeCore config file
type EdgeCoreConfig struct {
	metav1.TypeMeta
	// DataBase indicates database info
	// +Required
	DataBase *DataBase `json:"database,omitempty"`
	// Modules indicates EdgeCore modules config
	// +Required
	Modules *Modules `json:"modules,omitempty"`
}

// DataBase indicates the database info
type DataBase struct {
	// DriverName indicates database driver name
	// default "sqlite3"
	DriverName string `json:"driverName,omitempty"`
	// AliasName indicates alias name
	// default "default"
	AliasName string `json:"aliasName,omitempty"`
	// DataSource indicates the data source path
	// default "/var/lib/kubeedge/edgecore.db"
	DataSource string `json:"dataSource,omitempty"`
}

// Modules indicates the modules which edgeCore will be used
type Modules struct {
	// Edged indicates edged module config
	// +Required
	Edged *Edged `json:"edged,omitempty"`
	// EdgeHub indicates edgeHub module config
	// +Required
	EdgeHub *EdgeHub `json:"edgeHub,omitempty"`
	// EventBus indicates eventBus config for edgeCore
	// +Required
	EventBus *EventBus `json:"eventBus,omitempty"`
	// MetaManager indicates meta module config
	// +Required
	MetaManager *MetaManager `json:"metaManager,omitempty"`
	// ServiceBus indicates serviceBus module config
	ServiceBus *ServiceBus `json:"serviceBus,omitempty"`
	// DeviceTwin indicates deviceTwin module config
	DeviceTwin *DeviceTwin `json:"deviceTwin,omitempty"`
	// DBTest indicates dbTest module config
	DBTest *DBTest `json:"dbTest,omitempty"`
	// EdgeMesh indicates edgeMesh module config
	// +Required
	EdgeMesh *EdgeMesh `json:"edgeMesh,omitempty"`
	// EdgeStream indicates edgestream module config
	// +Required
	EdgeStream *EdgeStream `json:"edgeStream,omitempty"`
}

// Edged indicates the config fo edged module
// edged is lighted-kubelet
type Edged struct {
	// Enable indicates whether edged is enabled,
	// if set to false (for debugging etc.), skip checking other edged configs.
	// default true
	Enable bool `json:"enable,omitempty"`
	// NodeStatusUpdateFrequency indicates node status update frequency (second)
	// default 10
	NodeStatusUpdateFrequency int32 `json:"nodeStatusUpdateFrequency,omitempty"`
	// RuntimeType indicates cri runtime ,support: docker, remote
	// default "docker"
	RuntimeType string `json:"runtimeType,omitempty"`
	// DockerAddress indicates docker server address
	// default "unix:///var/run/docker.sock"
	DockerAddress string `json:"dockerAddress,omitempty"`
	// RemoteRuntimeEndpoint indicates remote runtime endpoint
	// default "unix:///var/run/dockershim.sock"
	RemoteRuntimeEndpoint string `json:"remoteRuntimeEndpoint,omitempty"`
	// RemoteImageEndpoint indicates remote image endpoint
	// default "unix:///var/run/dockershim.sock"
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
	// RegisterNode enables automatic registration
	// default true
	RegisterNode bool `json:"registerNode,omitempty"`
	//RegisterNodeNamespace indicates register node namespace
	// default "default"
	RegisterNodeNamespace string `json:"registerNodeNamespace,omitempty"`
	// InterfaceName indicates interface name
	// default "eth0"
	// DEPRECATED after v1.5
	InterfaceName string `json:"interfaceName,omitempty"`
	// ConcurrentConsumers indicates concurrent consumers for pod add or remove operation
	// default 5
	ConcurrentConsumers int `json:"concurrentConsumers,omitempty"`
	// DevicePluginEnabled indicates enable device plugin
	// default false
	// Note: Can not use "omitempty" option, it will affect the output of the default configuration file
	DevicePluginEnabled bool `json:"devicePluginEnabled"`
	// GPUPluginEnabled indicates enable gpu plugin
	// default false,
	// Note: Can not use "omitempty" option, it will affect the output of the default configuration file
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
	// CGroupDriver indicates container cgroup driver, support: cgroupfs, systemd
	// default "cgroupfs"
	// +Required
	CGroupDriver string `json:"cgroupDriver,omitempty"`
	// NetworkPluginName indicates the name of the network plugin to be invoked,
	// if an empty string is specified, use noop plugin
	// default ""
	NetworkPluginName string `json:"networkPluginName,omitempty"`
	// CNIConfDir indicates the full path of the directory in which to search for CNI config files
	// default "/etc/cni/net.d"
	CNIConfDir string `json:"cniConfDir,omitempty"`
	// CNIBinDir indicates a comma-separated list of full paths of directories
	// in which to search for CNI plugin binaries
	// default "/opt/cni/bin"
	CNIBinDir string `json:"cniBinDir,omitempty"`
	// CNICacheDir indicates the full path of the directory in which CNI should store cache files
	// default "/var/lib/cni/cache"
	CNICacheDir string `json:"cniCacheDirs,omitempty"`
	// NetworkPluginMTU indicates the MTU to be passed to the network plugin
	// default 1500
	NetworkPluginMTU int32 `json:"networkPluginMTU,omitempty"`
	// CgroupsPerQOS enables QoS based Cgroup hierarchy: top level cgroups for QoS Classes
	// And all Burstable and BestEffort pods are brought up under their
	// specific top level QoS cgroup.
	// Default: true
	CgroupsPerQOS bool `json:"cgroupsPerQOS"`
	// CgroupRoot is the root cgroup to use for pods.
	// If CgroupsPerQOS is enabled, this is the root of the QoS cgroup hierarchy.
	// Default: ""
	CgroupRoot string `json:"cgroupRoot"`
	// EdgeCoreCgroups is the absolute name of cgroups to isolate the edgecore in
	// Dynamic Kubelet Config (beta): This field should not be updated without a full node
	// reboot. It is safest to keep this value the same as the local config.
	// Default: ""
	EdgeCoreCgroups string `json:"edgeCoreCgroups,omitempty"`
	// systemCgroups is absolute name of cgroups in which to place
	// all non-kernel processes that are not already in a container. Empty
	// for no container. Rolling back the flag requires a reboot.
	// Dynamic Kubelet Config (beta): This field should not be updated without a full node
	// reboot. It is safest to keep this value the same as the local config.
	// Default: ""
	SystemCgroups string `json:"systemCgroups,omitempty"`
	// How frequently to calculate and cache volume disk usage for all pods
	// Dynamic Kubelet Config (beta): If dynamically updating this field, consider that
	// shortening the period may carry a performance impact.
	// Default: "1m"
	VolumeStatsAggPeriod time.Duration `json:"volumeStatsAggPeriod,omitempty"`
	// EnableMetrics indicates whether enable the metrics
	// default true
	EnableMetrics bool `json:"enableMetrics,omitempty"`
}

// EdgeHub indicates the EdgeHub module config
type EdgeHub struct {
	// Enable indicates whether EdgeHub is enabled,
	// if set to false (for debugging etc.), skip checking other EdgeHub configs.
	// default true
	Enable bool `json:"enable,omitempty"`
	// Heartbeat indicates heart beat (second)
	// default 15
	Heartbeat int32 `json:"heartbeat,omitempty"`
	// ProjectID indicates project id
	// default e632aba927ea4ac2b575ec1603d56f10
	ProjectID string `json:"projectID,omitempty"`
	// TLSCAFile set ca file path
	// default "/etc/kubeedge/ca/rootCA.crt"
	TLSCAFile string `json:"tlsCaFile,omitempty"`
	// TLSCertFile indicates the file containing x509 Certificate for HTTPS
	// default "/etc/kubeedge/certs/server.crt"
	TLSCertFile string `json:"tlsCertFile,omitempty"`
	// TLSPrivateKeyFile indicates the file containing x509 private key matching tlsCertFile
	// default "/etc/kubeedge/certs/server.key"
	TLSPrivateKeyFile string `json:"tlsPrivateKeyFile,omitempty"`
	// Quic indicates quic config for EdgeHub module
	// Optional if websocket is configured
	Quic *EdgeHubQUIC `json:"quic,omitempty"`
	// WebSocket indicates websocket config for EdgeHub module
	// Optional if quic is configured
	WebSocket *EdgeHubWebSocket `json:"websocket,omitempty"`
	// Token indicates the priority of joining the cluster for the edge
	Token string `json:"token"`
	// HTTPServer indicates the server for edge to apply for the certificate.
	HTTPServer string `json:"httpServer,omitempty"`
	// RotateCertificates indicates whether edge certificate can be rotated
	// default true
	RotateCertificates bool `json:"rotateCertificates,omitempty"`
}

// EdgeHubQUIC indicates the quic client config
type EdgeHubQUIC struct {
	// Enable indicates whether enable this protocol
	// default false
	Enable bool `json:"enable,omitempty"`
	// HandshakeTimeout indicates hand shake timeout (second)
	// default 30
	HandshakeTimeout int32 `json:"handshakeTimeout,omitempty"`
	// ReadDeadline indicates read dead line (second)
	// default 15
	ReadDeadline int32 `json:"readDeadline,omitempty"`
	// Server indicates quic server address (ip:port)
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
	// Enable indicates whether EventBus is enabled, if set to false (for debugging etc.),
	// skip checking other EventBus configs.
	// default true
	Enable bool `json:"enable,omitempty"`
	// MqttQOS indicates mqtt qos
	// 0: QOSAtMostOnce, 1: QOSAtLeastOnce, 2: QOSExactlyOnce
	// default 0
	// Note: Can not use "omitempty" option,  It will affect the output of the default configuration file
	MqttQOS uint8 `json:"mqttQOS"`
	// MqttRetain indicates whether server will store the message and can be delivered to future subscribers,
	// if this flag set true, sever will store the message and can be delivered to future subscribers
	// default false
	// Note: Can not use "omitempty" option,  It will affect the output of the default configuration file
	MqttRetain bool `json:"mqttRetain"`
	// MqttSessionQueueSize indicates the size of how many sessions will be handled.
	// default 100
	MqttSessionQueueSize int32 `json:"mqttSessionQueueSize,omitempty"`
	// MqttServerInternal indicates internal mqtt broker url
	// default "tcp://127.0.0.1:1884"
	MqttServerInternal string `json:"mqttServerInternal,omitempty"`
	// MqttServerExternal indicates external mqtt broker url
	// default "tcp://127.0.0.1:1883"
	MqttServerExternal string `json:"mqttServerExternal,omitempty"`
	// MqttMode indicates which broker type will be choose
	// 0: internal mqtt broker enable only.
	// 1: internal and external mqtt broker enable.
	// 2: external mqtt broker enable only
	// +Required
	// default: 2
	MqttMode MqttMode `json:"mqttMode"`
	// Tls indicates tls config for EventBus module
	TLS *EventBusTLS `json:"eventBusTLS,omitempty"`
}

// EventBusTLS indicates the EventBus tls config with MQTT broker
type EventBusTLS struct {
	// Enable indicates whether enable tls connection
	// default false
	Enable bool `json:"enable,omitempty"`
	// TLSMqttCAFile sets ca file path
	// default "/etc/kubeedge/ca/rootCA.crt"
	TLSMqttCAFile string `json:"tlsMqttCAFile,omitempty"`
	// TLSMqttCertFile indicates the file containing x509 Certificate for HTTPS
	// default "/etc/kubeedge/certs/server.crt"
	TLSMqttCertFile string `json:"tlsMqttCertFile,omitempty"`
	// TLSMqttPrivateKeyFile indicates the file containing x509 private key matching tlsMqttCertFile
	// default "/etc/kubeedge/certs/server.key"
	TLSMqttPrivateKeyFile string `json:"tlsMqttPrivateKeyFile,omitempty"`
}

// MetaManager indicates the MetaManager module config
type MetaManager struct {
	// Enable indicates whether MetaManager is enabled,
	// if set to false (for debugging etc.), skip checking other MetaManager configs.
	// default true
	Enable bool `json:"enable,omitempty"`
	// ContextSendGroup indicates send group
	ContextSendGroup metaconfig.GroupName `json:"contextSendGroup,omitempty"`
	// ContextSendModule indicates send module
	ContextSendModule metaconfig.ModuleName `json:"contextSendModule,omitempty"`
	// PodStatusSyncInterval indicates pod status sync
	// default 60
	PodStatusSyncInterval int32 `json:"podStatusSyncInterval,omitempty"`
	// RemoteQueryTimeout indicates remote query timeout (second)
	// default 60
	RemoteQueryTimeout int32 `json:"remoteQueryTimeout,omitempty"`
}

// ServiceBus indicates the ServiceBus module config
type ServiceBus struct {
	// Enable indicates whether ServiceBus is enabled,
	// if set to false (for debugging etc.), skip checking other ServiceBus configs.
	// default false
	Enable bool `json:"enable"`
}

// DeviceTwin indicates the DeviceTwin module config
type DeviceTwin struct {
	// Enable indicates whether DeviceTwin is enabled,
	// if set to false (for debugging etc.), skip checking other DeviceTwin configs.
	// default true
	Enable bool `json:"enable,omitempty"`
}

// DBTest indicates the DBTest module config
type DBTest struct {
	// Enable indicates whether DBTest is enabled,
	// if set to false (for debugging etc.), skip checking other DBTest configs.
	// default false
	Enable bool `json:"enable"`
}

// EdgeMesh indicates the EdgeMesh module config
type EdgeMesh struct {
	// Enable indicates whether EdgeMesh is enabled,
	// if set to false (for debugging etc.), skip checking other EdgeMesh configs.
	// default true
	Enable bool `json:"enable,omitempty"`
	// lbStrategy indicates load balance strategy name
	// default "RoundRobin"
	LBStrategy string `json:"lbStrategy,omitempty"`
	// ListenInterface indicates the listen interface of EdgeMesh
	// default "docker0"
	ListenInterface string `json:"listenInterface,omitempty"`
	// SubNet indicates the subnet of EdgeMesh
	// default "9.251.0.0/16"
	SubNet string `json:"subNet,omitempty"`
	// ListenPort indicates the listen port of EdgeMesh
	// default 40001
	ListenPort int `json:"listenPort,omitempty"`
}

// EdgeSream indicates the stream controller
type EdgeStream struct {
	// Enable indicates whether edgestream is enabled, if set to false (for debugging etc.), skip checking other configs.
	// default true
	Enable bool `json:"enable"`

	// TLSTunnelCAFile indicates ca file path
	// default /etc/kubeedge/ca/rootCA.crt
	TLSTunnelCAFile string `json:"tlsTunnelCAFile,omitempty"`

	// TLSTunnelCertFile indicates the file containing x509 Certificate for HTTPS
	// default /etc/kubeedge/certs/server.crt
	TLSTunnelCertFile string `json:"tlsTunnelCertFile,omitempty"`
	// TLSTunnelPrivateKeyFile indicates the file containing x509 private key matching tlsCertFile
	// default /etc/kubeedge/certs/server.key
	TLSTunnelPrivateKeyFile string `json:"tlsTunnelPrivateKeyFile,omitempty"`

	// HandshakeTimeout indicates handshake timeout (second)
	// default  30
	HandshakeTimeout int32 `json:"handshakeTimeout,omitempty"`
	// ReadDeadline indicates read dead line (second)
	// default 15
	ReadDeadline int32 `json:"readDeadline,omitempty"`
	// TunnelServer indicates websocket server address (ip:port)
	// +Required
	TunnelServer string `json:"server,omitempty"`
	// WriteDeadline indicates write dead line (second)
	// default 15
	WriteDeadline int32 `json:"writeDeadline,omitempty"`
}
