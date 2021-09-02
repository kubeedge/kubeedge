---
title: KubeEdge Component Config Proposal

authors:
  - "@kadisi"
  - "@fisherxu"

approvers:
  - "@kevin-wangzefeng"
  - "@sids-b"

creation-date: 2019-10-02

status: implemented

---
* [KubeEdge Component Config Proposal](#kubeedge-component-config-proposal)
   * [Terminology](#terminology)
   * [Proposal](#proposal)
   * [Goals](#goals)
   * [Principle](#principle)
   * [How to do](#how-to-do)
      * [KubeEdge component config apis definition](#kubeedge-component-config-apis-definition)
         * [meta config apis](#meta-config-apis)
         * [cloudcore config apis](#cloudcore-config-apis)
         * [edgecore config apis](#edgecore-config-apis)
         * [edgesite config apis](#edgesite-config-apis)
      * [Default config file locations](#default-config-file-locations)
      * [Use --defaultconfig flag to generate full component config with default values](#use---defaultconfig-flag-to-generate-full-component-config-with-default-values)
      * [Compatible with old configuration files](#compatible-with-old-configuration-files)
      * [new config file need version number](#new-config-file-need-version-number)
      * [How to pass the configuration to each module](#how-to-pass-the-configuration-to-each-module)
      * [Use keadm to install and configure KubeEdge components](#use-keadm-to-install-and-configure-kubeedge-components)
   * [Task list tracking](#task-list-tracking)

# KubeEdge Component Config Proposal

## Terminology

* **KubeEdge components:** refers to binaries e.g. cloudcore, admission, edgecore, edgesite, etc.

* **KubeEdge modules:** refers to modules e.g. cloudhub, edgecontroller, devicecontroller, devicetwin, edged, edgehub, eventbus, metamanager, servicebus, etc.

## Proposal

Currently, KubeEdge components' configuration files are in the conf directory at the same level and have 3 configuration files, it is difficult to maintain and extend.

KubeEdge uses beehive package to analyse configuration files, when the program is running, it will print a lot of logs first. When we add subcommands to the program, such as `--version` , it will still print a lot of configuration information and then output the version information.

We recommend referring to the kubernetes component config api design to redesign the definition of the KubeEdge component configuration file:

[kubelet config file](https://kubernetes.io/docs/tasks/administer-cluster/kubelet-config-file/)

[kubelet api config definition](https://github.com/kubernetes/kubernetes/blob/master/pkg/kubelet/apis/config/types.go)

## Goals

* KubeEdge components use one configuration file instead of the original 3 configuration files. It support json or yaml format, defaut is yaml.

* Start the KubeEdge component with the --config flag, this flag set to the path of the component's config file. The component will then load its config from this file, if --config flag not set, component will read a default configuration file.

* Configuration file's definition refers to the kubernetes component config api design, which needs to be with a version number for future version management.

* Need to abstract the apis for KubeEdge component configuration file  and defined in `pkg/apis/{components}/` dir.

* keadm uses the KubeEdge component config api to generate  configuration file for each component, and allows additional command line flags to override the configuration of each component. This will make it easier to install and configure KubeEdge components.

* After KubeEdge component starts, it will first load its config from configfile, verifies the legality, and then passes the corresponding config to the KubeEdge modules through the Register method of each module.

* New configuration files should consider backward compatibility in future upgrades

* Support conversion of 3 old configfiles to one new configfile.

  take cloudcore as an example: now cloudcore has 3 configfiles: `controller.yaml,logging.yaml,modules.yaml`, We need to convert those three old configuration files into one new configuration file in one way.


## Principle

* **Backward compatibility**

	`keadm` provides subcommands for conversion

* **Forward compatibility**

	For configuration file, support addition/depreciation of some fields, **Modify field not allowed**.
	Configuration need a version field.


## How to do

### KubeEdge component config apis definition

#### meta config apis

defined in `pkg/apis/meta/v1alpha1/types.go`

```go

package v1alpha1

type ModuleName string
type GroupName string

// Available modules for CloudCore
const (
	ModuleNameEdgeController   ModuleName = "edgecontroller"
	ModuleNameDeviceController ModuleName = "devicecontroller"
	ModuleNameCloudHub         ModuleName = "cloudhub"
)

// Available modules for EdgeCore
const (
	ModuleNameEventBus   ModuleName = "eventbus"
	ModuleNameServiceBus ModuleName = "servicebus"
	// TODO @kadisi change websocket to edgehub
	ModuleNameEdgeHub     ModuleName = "websocket"
	ModuleNameMetaManager ModuleName = "metaManager"
	ModuleNameEdged       ModuleName = "edged"
	ModuleNameTwin        ModuleName = "twin"
	ModuleNameDBTest      ModuleName = "dbTest"
	ModuleNameEdgeMesh    ModuleName = "edgemesh"
)

// Available modules group
const (
	GroupNameHub            GroupName = "hub"
	GroupNameEdgeController GroupName = "edgecontroller"
	GroupNameBus            GroupName = "bus"
	GroupNameTwin           GroupName = "twin"
	GroupNameMeta           GroupName = "meta"
	GroupNameEdged          GroupName = "edged"
	GroupNameUser           GroupName = "user"
	GroupNameMesh           GroupName = "mesh"
)


```

#### cloudcore config apis

defined in `pkg/apis/cloudcore/v1alpha1/types.go`

```go

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	metaconfig "github.com/kubeedge/kubeedge/pkg/apis/componentconfig/meta/v1alpha1"
)

// CloudCoreConfig indicates the config of cloudcore which get from cloudcore config file
type CloudCoreConfig struct {
	metav1.TypeMeta
	// KubeAPIConfig indicates the kubernetes cluster info which cloudcore will connected
	// +Required
	KubeAPIConfig *KubeAPIConfig `json:"kubeAPIConfig,omitempty"`
	// Modules indicates cloudcore modules config
	// +Required
	Modules *Modules `json:"modules,omitempty"`
}

// KubeAPIConfig indicates the configuration for interacting with k8s server
type KubeAPIConfig struct {
	// Master indicates the address of the Kubernetes API server (overrides any value in KubeConfig)
	// such as https://127.0.0.1:8443
	// default ""
	// Note: Can not use "omitempty" option,  It will affect the output of the default configuration file
	Master string `json:"master"`
	// ContentType indicates the ContentType of message transmission when interacting with k8s
	// default application/vnd.kubernetes.protobuf
	ContentType string `json:"contentType,omitempty"`
	// QPS to while talking with kubernetes apiserve
	// default 100
	QPS int32 `json:"qps,omitempty"`
	// Burst to use while talking with kubernetes apiserver
	// default 200
	Burst int32 `json:"burst,omitempty"`
	// KubeConfig indicates the path to kubeConfig file with authorization and master location information.
	// default "/root/.kube/config"
	// +Required
	KubeConfig string `json:"kubeConfig"`
}

// Modules indicates the modules of cloudCore will be use
type Modules struct {
	// CloudHub indicates CloudHub module config
	CloudHub *CloudHub `json:"cloudHub,omitempty"`
	// EdgeController indicates edgeController module config
	EdgeController *EdgeController `json:"edgeController,omitempty"`
	// DeviceController indicates deviceController module config
	DeviceController *DeviceController `json:"deviceController,omitempty"`
}

// CloudHub indicates the config of CloudHub module.
// CloudHub is a web socket or quic server responsible for watching changes at the cloud side,
// caching and sending messages to EdgeHub.
type CloudHub struct {
	// Enable indicates whether CloudHub is enabled, if set to false (for debugging etc.),
	// skip checking other CloudHub configs.
	// default true
	Enable bool `json:"enable,omitempty"`
	// KeepaliveInterval indicates keep-alive interval (second)
	// default 30
	KeepaliveInterval int32 `json:"keepaliveInterval,omitempty"`
	// NodeLimit indicates node limit
	// default 10
	NodeLimit int32 `json:"nodeLimit,omitempty"`
	// TLSCAFile indicates ca file path
	// default "/etc/kubeedge/ca/rootCA.crt"
	TLSCAFile string `json:"tlsCAFile,omitempty"`
	// TLSCertFile indicates cert file path
	// default "/etc/kubeedge/certs/edge.crt"
	TLSCertFile string `json:"tlsCertFile,omitempty"`
	// TLSPrivateKeyFile indicates key file path
	// default "/etc/kubeedge/certs/edge.key"
	TLSPrivateKeyFile string `json:"tlsPrivateKeyFile,omitempty"`
	// WriteTimeout indicates write time (second)
	// default 30
	WriteTimeout int32 `json:"writeTimeout,omitempty"`
	// Quic indicates quic server info
	Quic *CloudHubQUIC `json:"quic,omitempty"`
	// UnixSocket set unixsocket server info
	UnixSocket *CloudHubUnixSocket `json:"unixsocket,omitempty"`
	// WebSocket indicates websocket server info
	// +Required
	WebSocket *CloudHubWebSocket `json:"websocket,omitempty"`
}

// CloudHubQUIC indicates the quic server config
type CloudHubQUIC struct {
	// Enable indicates whether enable quic protocol
	// default false
	Enable bool `json:"enable,omitempty"`
	// Address set server ip address
	// default 0.0.0.0
	Address string `json:"address,omitempty"`
	// Port set open port for quic server
	// default 10001
	Port uint32 `json:"port,omitempty"`
	// MaxIncomingStreams set the max incoming stream for quic server
	// default 10000
	MaxIncomingStreams int32 `json:"maxIncomingStreams,omitempty"`
}

// CloudHubUnixSocket indicates the unix socket config
type CloudHubUnixSocket struct {
	// Enable indicates whether enable unix domain socket protocol
	// default true
	Enable bool `json:"enable,omitempty"`
	// Address indicates unix domain socket address
	// default "unix:///var/lib/kubeedge/kubeedge.sock"
	Address string `json:"address,omitempty"`
}

// CloudHubWebSocket indicates the websocket config of CloudHub
type CloudHubWebSocket struct {
	// Enable indicates whether enable websocket protocol
	// default true
	Enable bool `json:"enable,omitempty"`
	// Address indicates server ip address
	// default 0.0.0.0
	Address string `json:"address,omitempty"`
	// Port indicates the open port for websocket server
	// default 10000
	Port uint32 `json:"port,omitempty"`
}

// EdgeController indicates the config of edgeController module
type EdgeController struct {
	// Enable indicates whether edgeController is enabled,
	// if set to false (for debugging etc.), skip checking other edgeController configs.
	// default true
	Enable bool `json:"enable,omitempty"`
	// NodeUpdateFrequency indicates node update frequency (second)
	// default 10
	NodeUpdateFrequency int32 `json:"nodeUpdateFrequency,omitempty"`
	// Buffer indicates k8s resource buffer
	Buffer *EdgeControllerBuffer `json:"buffer,omitempty"`
	// Context indicates send,receive,response modules for edgeController module
	Context *EdgeControllerContext `json:"context,omitempty"`
	// Load indicates edgeController load
	Load *EdgeControllerLoad `json:"load,omitempty"`
}

// EdgeControllerBuffer indicates the edgeController buffer
type EdgeControllerBuffer struct {
	// UpdatePodStatus indicates the buffer of pod status
	// default 1024
	UpdatePodStatus int32 `json:"updatePodStatus,omitempty"`
	// UpdateNodeStatus indicates the buffer of update node status
	// default 1024
	UpdateNodeStatus int32 `json:"updateNodeStatus,omitempty"`
	// QueryConfigMap indicates the buffer of query configMap
	// default 1024
	QueryConfigMap int32 `json:"queryConfigMap,omitempty"`
	// QuerySecret indicates the buffer of query secret
	// default 1024
	QuerySecret int32 `json:"querySecret,omitempty"`
	// QueryService indicates the buffer of query service
	// default 1024
	QueryService int32 `json:"queryService,omitempty"`
	// QueryEndpoints indicates the buffer of query endpoint
	// default 1024
	QueryEndpoints int32 `json:"queryEndpoints,omitempty"`
	// PodEvent indicates the buffer of pod event
	// default 1
	PodEvent int32 `json:"podEvent,omitempty"`
	// ConfigMapEvent indicates the buffer of configMap event
	// default 1
	ConfigMapEvent int32 `json:"configMapEvent,omitempty"`
	// SecretEvent indicates the buffer of secret event
	// default 1
	SecretEvent int32 `json:"secretEvent,omitempty"`
	// ServiceEvent indicates the buffer of service event
	// default 1
	ServiceEvent int32 `json:"serviceEvent,omitempty"`
	// EndpointsEvent indicates the buffer of endpoint event
	// default 1
	EndpointsEvent int32 `json:"endpointsEvent,omitempty"`
	// QueryPersistentVolume indicates the buffer of query persistent volume
	// default 1024
	QueryPersistentVolume int32 `json:"queryPersistentVolume,omitempty"`
	// QueryPersistentVolumeClaim indicates the buffer of query persistent volume claim
	// default 1024
	QueryPersistentVolumeClaim int32 `json:"queryPersistentVolumeClaim,omitempty"`
	// QueryVolumeAttachment indicates the buffer of query volume attachment
	// default 1024
	QueryVolumeAttachment int32 `json:"queryVolumeAttachment,omitempty"`
	// QueryNode indicates the buffer of query node
	// default 1024
	QueryNode int32 `json:"queryNode,omitempty"`
	// UpdateNode indicates the buffer of update node
	// default 1024
	UpdateNode int32 `json:"updateNode,omitempty"`
	// DeletePod indicates the buffer of delete pod message from edge
	// default 1024
	DeletePod int32 `json:"deletePod,omitempty"`
}

// EdgeControllerContext indicates the edgeController context
type EdgeControllerContext struct {
	// SendModule indicates which module will send message to
	SendModule metaconfig.ModuleName `json:"sendModule,omitempty"`
	// ReceiveModule indicates which module will receive message from
	ReceiveModule metaconfig.ModuleName `json:"receiveModule,omitempty"`
	// ResponseModule indicates which module will response message to
	ResponseModule metaconfig.ModuleName `json:"responseModule,omitempty"`
}

// EdgeControllerLoad indicates the edgeController load
type EdgeControllerLoad struct {
	// UpdatePodStatusWorkers indicates the load of update pod status workers
	// default 1
	UpdatePodStatusWorkers int32 `json:"updatePodStatusWorkers,omitempty"`
	// UpdateNodeStatusWorkers indicates the load of update node status workers
	// default 1
	UpdateNodeStatusWorkers int32 `json:"updateNodeStatusWorkers,omitempty"`
	// QueryConfigMapWorkers indicates the load of query config map workers
	// default 1
	QueryConfigMapWorkers int32 `json:"queryConfigMapWorkers,omitempty"`
	// QuerySecretWorkers indicates the load of query secret workers
	// default 4
	QuerySecretWorkers int32 `json:"querySecretWorkers,omitempty"`
	// QueryServiceWorkers indicates the load of query service workers
	// default 4
	QueryServiceWorkers int32 `json:"queryServiceWorkers,omitempty"`
	// QueryEndpointsWorkers indicates the load of query endpoint workers
	// default 4
	QueryEndpointsWorkers int32 `json:"queryEndpointsWorkers,omitempty"`
	// QueryPersistentVolumeWorkers indicates the load of query persistent volume workers
	// default 4
	QueryPersistentVolumeWorkers int32 `json:"queryPersistentVolumeWorkers,omitempty"`
	// QueryPersistentVolumeClaimWorkers indicates the load of query persistent volume claim workers
	// default 4
	QueryPersistentVolumeClaimWorkers int32 `json:"queryPersistentVolumeClaimWorkers,omitempty"`
	// QueryVolumeAttachmentWorkers indicates the load of query volume attachment workers
	// default 4
	QueryVolumeAttachmentWorkers int32 `json:"queryVolumeAttachmentWorkers,omitempty"`
	// QueryNodeWorkers indicates the load of query node workers
	// default 4
	QueryNodeWorkers int32 `json:"queryNodeWorkers,omitempty"`
	// UpdateNodeWorkers indicates the load of update node workers
	// default 4
	UpdateNodeWorkers int32 `json:"updateNodeWorkers,omitempty"`
	// DeletePodWorkers indicates the load of delete pod workers
	// default 4
	DeletePodWorkers int32 `json:"deletePodWorkers,omitempty"`
}

// DeviceController indicates the device controller
type DeviceController struct {
	// Enable indicates whether deviceController is enabled,
	// if set to false (for debugging etc.), skip checking other deviceController configs.
	// default true
	Enable bool `json:"enable,omitempty"`
	// Context indicates send,receive,response modules for deviceController module
	Context *DeviceControllerContext `json:"context,omitempty"`
	// Buffer indicates Device controller buffer
	Buffer *DeviceControllerBuffer `json:"buffer,omitempty"`
	// Load indicates DeviceController Load
	Load *DeviceControllerLoad `json:"load,omitempty"`
}

// DeviceControllerContext indicates the device controller context
type DeviceControllerContext struct {
	// SendModule indicates which module will send message to
	SendModule metaconfig.ModuleName `json:"sendModule,omitempty"`
	// ReceiveModule indicates which module will receive message from
	ReceiveModule metaconfig.ModuleName `json:"receiveModule,omitempty"`
	// ResponseModule indicates which module will response message to
	ResponseModule metaconfig.ModuleName `json:"responseModule,omitempty"`
}

// DeviceControllerBuffer indicates deviceController buffer
type DeviceControllerBuffer struct {
	// UpdateDeviceStatus indicates the buffer of update device status
	// default 1024
	UpdateDeviceStatus int32 `json:"updateDeviceStatus,omitempty"`
	// DeviceEvent indicates the buffer of device event
	// default 1
	DeviceEvent int32 `json:"deviceEvent,omitempty"`
	// DeviceModelEvent indicates the buffer of device model event
	// default 1
	DeviceModelEvent int32 `json:"deviceModelEvent,omitempty"`
}

// DeviceControllerLoad indicates the deviceController load
type DeviceControllerLoad struct {
	// UpdateDeviceStatusWorkers indicates the load of update device status workers
	// default 1
	UpdateDeviceStatusWorkers int32 `json:"updateDeviceStatusWorkers,omitempty"`
}
```

#### edgecore config apis

defined in `pkg/apis/edgecore/v1alpha1/types.go`

```go

package v1alpha1

import (
	metaconfig "github.com/kubeedge/kubeedge/pkg/apis/componentconfig/meta/v1alpha1"
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
	// default "/var/lib/kubeedge/edge.db"
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
	// Note: Can not use "omitempty" option, it will affect the output of the default configuration file
	// +Required
	ClusterDNS string `json:"clusterDNS"`
	// ClusterDomain indicates cluster domain
	// Note: Can not use "omitempty" option, it will affect the output of the default configuration file
	ClusterDomain string `json:"clusterDomain"`
	// EdgedMemoryCapacity indicates memory capacity (byte)
	// default 7852396000
	EdgedMemoryCapacity int64 `json:"edgedMemoryCapacity,omitempty"`
	// PodSandboxImage is the image whose network/ipc namespaces containers in each pod will use.
	// +Required
	// default kubeedge/pause
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
	// default cgroupfs
	// +Required
	CGroupDriver string `json:"cgroupDriver,omitempty"`
	// RegisterNode enables automatic registration
	// default true
	RegisterNode bool `json:"registerNode,omitempty"`
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
	// default /etc/kubeedge/ca/rootCA.crt
	TLSCAFile string `json:"tlsCaFile,omitempty"`
	// TLSCertFile indicates the file containing x509 Certificate for HTTPS
	// default /etc/kubeedge/certs/edge.crt
	TLSCertFile string `json:"tlsCertFile,omitempty"`
	// TLSPrivateKeyFile indicates the file containing x509 private key matching tlsCertFile
	// default /etc/kubeedge/certs/edge.key
	TLSPrivateKeyFile string `json:"tlsPrivateKeyFile,omitempty"`
	// Quic indicates quic config for EdgeHub module
	// Optional if websocket is configured
	Quic *EdgeHubQUIC `json:"quic,omitempty"`
	// WebSocket indicates websocket config for EdgeHub module
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
	// Enable indicates whether EventBus is enabled,
	// if set to false (for debugging etc.), skip checking other EventBus configs.
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
	// 0: internal mqtt broker enable only.
	// 1: internal and external mqtt broker enable.
	// 2: external mqtt broker enable only
	// +Required
	// default: 2
	MqttMode MqttMode `json:"mqttMode"`
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
	PodStatusSyncInterval int32 `json:"podStatusSyncInterval,omitempty"`
}

// ServiceBus indicates the ServiceBus module config
type ServiceBus struct {
	// Enable indicates whether ServiceBus is enabled,
	// if set to false (for debugging etc.), skip checking other ServiceBus configs.
	// default true
	Enable bool `json:"enable,omitempty"`
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
	LBStrategy string `json:"lbStrategy,omitempty"`
}

```

#### edgesite config apis

defined in `pkg/apis/edgesite/v1alpha1/types.go`

```go

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	cloudcoreconfig "github.com/kubeedge/kubeedge/pkg/apis/componentconfig/cloudcore/v1alpha1"
	edgecoreconfig "github.com/kubeedge/kubeedge/pkg/apis/componentconfig/edgecore/v1alpha1"
)

const (
	// DataBaseDriverName is sqlite3
	DataBaseDriverName = "sqlite3"
	// DataBaseAliasName is default
	DataBaseAliasName = "default"
	// DataBaseDataSource is edge.db
	DataBaseDataSource = "/var/lib/kubeedge/edgesite.db"
)

// EdgeSiteConfig indicates the edgesite config which read from edgesite config file
type EdgeSiteConfig struct {
	metav1.TypeMeta
	// DataBase indicates database info
	// +Required
	DataBase *edgecoreconfig.DataBase `json:"database,omitempty"`
	// KubeAPIConfig indicates the kubernetes cluster info which cloudcore will connected
	// +Required
	KubeAPIConfig *cloudcoreconfig.KubeAPIConfig `json:"kubeAPIConfig,omitempty"`
	// Modules indicates cloudcore modules config
	// +Required
	Modules *Modules `json:"modules,omitempty"`
}

// Modules indicates the modules which edgesite will be used
type Modules struct {
	// EdgeController indicates edgecontroller module config
	EdgeController *cloudcoreconfig.EdgeController `json:"edgeController,omitempty"`
	// Edged indicates edged module config
	// +Required
	Edged *edgecoreconfig.Edged `json:"edged,omitempty"`
	// MetaManager indicates meta module config
	// +Required
	MetaManager *edgecoreconfig.MetaManager `json:"metaManager,omitempty"`
}

```

### Default config file locations

KubeEdge components would load config files in path `/etc/kubeedge/config/` by default, and users can customize the locations with `--config` flag:

* **cloudcore**

default load  `/etc/kubeedge/config/cloudcore.yaml` configfile
start cloudcore with specific config file location
`cloudcore --config "/<your-path-to-cloudcore-config>/cloudcore.yaml"`

* **edgecore**

default load  `/etc/kubeedge/config/edgecore.yaml` configfile

* **edgeside**

default load `/etc/kubeedge/config/edgesite.yaml` configfile

### Use `--defaultconfig` and `--minconfig` flag to generate default full and common config component config with default values


 With `--dfaultconfig` flag, users can easily get a default full config file as reference, with all fields (and field descriptions) included and default values set. It's useful to users that are new to KubeEdge, and they can modify/create their own configs accordingly. Because it is a full configuration, it is more suitable for advanced users.

 With `--minconfig` flag, users can easily get min used configurations as reference. It's useful to users that are new to KubeEdge, and they can modify/create their own configs accordingly. This configuration is suitable for beginners.

* cloudcore

`# cloudcore --defaultconfig`

```yaml

apiVersion: cloudcore.config.kubeedge.io/v1alpha1
kind: CloudCore
kubeAPIConfig:
  burst: 200
  contentType: application/vnd.kubernetes.protobuf
  kubeConfig: /root/.kube/config
  master: ""
  qps: 100
modules:
  cloudhub:
    enable: true
    keepaliveInterval: 30
    nodeLimit: 10
    quic:
      address: 0.0.0.0
      maxIncomingStreams: 10000
      port: 10001
    tlsCAFile: /etc/kubeedge/ca/rootCA.crt
    tlsCertFile: /etc/kubeedge/certs/edge.crt
    tlsPrivateKeyFile: /etc/kubeedge/certs/edge.key
    unixsocket:
      address: unix:///var/lib/kubeedge/kubeedge.sock
      enable: true
    websocket:
      address: 0.0.0.0
      enable: true
      port: 10000
    writeTimeout: 30
  devicecontroller:
    buffer:
      deviceEvent: 1
      deviceModelEvent: 1
      updateDeviceStatus: 1024
    context:
      receiveModule: devicecontroller
      responseModule: cloudhub
      sendModule: cloudhub
    enable: true
    load:
      updateDeviceStatusWorkers: 1
  edgecontroller:
    buffer:
      configmapEvent: 1
      endpointsEvent: 1
      podEvent: 1
      queryConfigMap: 1024
      queryEndpoints: 1024
      queryNode: 1024
      queryPersistentVolume: 1024
      queryPersistentVolumeClaim: 1024
      querySecret: 1024
      queryService: 1024
      queryVolumeAttachment: 1024
      secretEvent: 1
      serviceEvent: 1
      updateNode: 1024
      updateNodeStatus: 1024
      updatePodStatus: 1024
    context:
      receiveModule: edgecontroller
      responseModule: cloudhub
      sendModule: cloudhub
    enable: true
    load:
      queryConfigMapWorkers: 4
      queryEndpointsWorkers: 4
      queryNodeWorkers: 4
      queryPersistentVolumeClaimWorkers: 4
      queryPersistentVolumeWorkers: 4
      querySecretWorkers: 4
      queryServiceWorkers: 4
      queryVolumeAttachmentWorkers: 4
      updateNodeStatusWorkers: 1
      updateNodeWorkers: 4
      updatePodStatusWorkers: 1
    nodeUpdateFrequency: 10


```


`# cloudcore --minconfig`

```yaml

apiVersion: cloudcore.config.kubeedge.io/v1alpha1
kind: CloudCore
kubeAPIConfig:
  kubeConfig: /root/.kube/config
  master: ""
modules:
  cloudhub:
    nodeLimit: 10
    tlsCAFile: /etc/kubeedge/ca/rootCA.crt
    tlsCertFile: /etc/kubeedge/certs/edge.crt
    tlsPrivateKeyFile: /etc/kubeedge/certs/edge.key
    unixsocket:
      address: unix:///var/lib/kubeedge/kubeedge.sock
      enable: true
    websocket:
      address: 0.0.0.0
      enable: true
      port: 10000


```


* edgecore

`# edgecore --defaultconfig`

```yaml

apiVersion: edgecore.config.kubeedge.io/v1alpha1
database:
  aliasName: default
  dataSource: /var/lib/kubeedge/edgecore.db
  driverName: sqlite3
kind: EdgeCore
modules:
  dbtest:
    enable: false
  devicetwin:
    enable: true
  edged:
    cgroupDriver: cgroupfs
    clusterDNS: ""
    clusterDomain: ""
    devicePluginEnabled: false
    dockerAddress: unix:///var/run/docker.sock
    edgedMemoryCapacity: 7852396000
    enable: true
    gpuPluginEnabled: false
    hostnameOverride: zhangjiedeMacBook-Pro.local
    imageGCHighThreshold: 80
    imageGCLowThreshold: 40
    imagePullProgressDeadline: 60
    interfaceName: eth0
    maximumDeadContainersPerPod: 1
    nodeIP: 192.168.4.3
    nodeStatusUpdateFrequency: 10
    podSandboxImage: kubeedge/pause:3.1
    registerNode: true
    registerNodeNamespace: default
    remoteImageEndpoint: unix:///var/run/dockershim.sock
    remoteRuntimeEndpoint: unix:///var/run/dockershim.sock
    runtimeRequestTimeout: 2
    runtimeType: docker
  edgehub:
    enable: true
    heartbeat: 15
    projectID: e632aba927ea4ac2b575ec1603d56f10
    quic:
      handshakeTimeout: 30
      readDeadline: 15
      server: 127.0.0.1:10001
      writeDeadline: 15
    tlsCaFile: /etc/kubeedge/ca/rootCA.crt
    tlsCertFile: /etc/kubeedge/certs/edge.crt
    tlsPrivateKeyFile: /etc/kubeedge/certs/edge.key
    websocket:
      enable: true
      handshakeTimeout: 30
      readDeadline: 15
      server: 127.0.0.1:10000
      writeDeadline: 15
  edgemesh:
    enable: true
    lbStrategy: RoundRobin
  eventbus:
    enable: true
    mqttMode: 2
    mqttQOS: 0
    mqttRetain: false
    mqttServerExternal: tcp://127.0.0.1:1883
    mqttServerInternal: tcp://127.0.0.1:1884
    mqttSessionQueueSize: 100
  metamanager:
    contextSendGroup: hub
    contextSendModule: websocket
    enable: true
    podStatusSyncInterval: 60
  servicebus:
    enable: false

```

`# edgecore --minconfig`

```yaml

apiVersion: edgecore.config.kubeedge.io/v1alpha1
database:
  dataSource: /var/lib/kubeedge/edgecore.db
kind: EdgeCore
modules:
  edged:
    cgroupDriver: cgroupfs
    clusterDNS: ""
    clusterDomain: ""
    devicePluginEnabled: false
    dockerAddress: unix:///var/run/docker.sock
    gpuPluginEnabled: false
    hostnameOverride: zhangjiedeMacBook-Pro.local
    interfaceName: eth0
    nodeIP: 192.168.4.3
    podSandboxImage: kubeedge/pause:3.1
    remoteImageEndpoint: unix:///var/run/dockershim.sock
    remoteRuntimeEndpoint: unix:///var/run/dockershim.sock
    runtimeType: docker
  edgehub:
    heartbeat: 15
    tlsCaFile: /etc/kubeedge/ca/rootCA.crt
    tlsCertFile: /etc/kubeedge/certs/edge.crt
    tlsPrivateKeyFile: /etc/kubeedge/certs/edge.key
    websocket:
      enable: true
      handshakeTimeout: 30
      readDeadline: 15
      server: 127.0.0.1:10000
      writeDeadline: 15
  eventbus:
    mqttMode: 2
    mqttQOS: 0
    mqttRetain: false
    mqttServerExternal: tcp://127.0.0.1:1883
    mqttServerInternal: tcp://127.0.0.1:1884


```



* edgesite

`# edgesite --defaultconfig`

```yaml

apiVersion: edgesite.config.kubeedge.io/v1alpha1
database:
  aliasName: default
  dataSource: /var/lib/kubeedge/edgesite.db
  driverName: sqlite3
kind: EdgeSite
kubeAPIConfig:
  burst: 200
  contentType: application/vnd.kubernetes.protobuf
  kubeConfig: /root/.kube/config
  master: ""
  qps: 100
modules:
  edgecontroller:
    buffer:
      configmapEvent: 1
      endpointsEvent: 1
      podEvent: 1
      queryConfigMap: 1024
      queryEndpoints: 1024
      queryNode: 1024
      queryPersistentVolume: 1024
      queryPersistentVolumeClaim: 1024
      querySecret: 1024
      queryService: 1024
      queryVolumeAttachment: 1024
      secretEvent: 1
      serviceEvent: 1
      updateNode: 1024
      updateNodeStatus: 1024
      updatePodStatus: 1024
    context:
      receiveModule: edgecontroller
      responseModule: metaManager
      sendModule: metaManager
    enable: true
    load:
      queryConfigMapWorkers: 4
      queryEndpointsWorkers: 4
      queryNodeWorkers: 4
      queryPersistentVolumeClaimWorkers: 4
      queryPersistentVolumeWorkers: 4
      querySecretWorkers: 4
      queryServiceWorkers: 4
      queryVolumeAttachmentWorkers: 4
      updateNodeStatusWorkers: 1
      updateNodeWorkers: 4
      updatePodStatusWorkers: 1
    nodeUpdateFrequency: 10
  edged:
    cgroupDriver: cgroupfs
    clusterDNS: ""
    clusterDomain: ""
    devicePluginEnabled: false
    dockerAddress: unix:///var/run/docker.sock
    edgedMemoryCapacity: 7852396000
    enable: true
    gpuPluginEnabled: false
    hostnameOverride: zhangjiedeMacBook-Pro.local
    imageGCHighThreshold: 80
    imageGCLowThreshold: 40
    imagePullProgressDeadline: 60
    interfaceName: eth0
    maximumDeadContainersPerPod: 1
    nodeIP: 192.168.4.3
    nodeStatusUpdateFrequency: 10
    podSandboxImage: kubeedge/pause:3.1
    registerNode: true
    registerNodeNamespace: default
    remoteImageEndpoint: unix:///var/run/dockershim.sock
    remoteRuntimeEndpoint: unix:///var/run/dockershim.sock
    runtimeRequestTimeout: 2
    runtimeType: docker
  metamanager:
    contextSendGroup: edgecontroller
    contextSendModule: edgecontroller
    enable: true
    podStatusSyncInterval: 60

```

`# edgesite --minconfig`

```yaml

apiVersion: edgesite.config.kubeedge.io/v1alpha1
database:
  dataSource: /var/lib/kubeedge/edgesite.db
kind: EdgeSite
kubeAPIConfig:
  kubeConfig: /root/.kube/config
  master: ""
modules:
  edged:
    cgroupDriver: cgroupfs
    clusterDNS: ""
    clusterDomain: ""
    devicePluginEnabled: false
    dockerAddress: unix:///var/run/docker.sock
    gpuPluginEnabled: false
    hostnameOverride: zhangjiedeMacBook-Pro.local
    interfaceName: eth0
    nodeIP: 192.168.4.3
    podSandboxImage: kubeedge/pause:3.1
    remoteImageEndpoint: unix:///var/run/dockershim.sock
    remoteRuntimeEndpoint: unix:///var/run/dockershim.sock
    runtimeType: docker

```


### Compatible with old configuration files

In order to support the old configuration file , there are 2 options:

1. KubeEdge components support the old configuration file at runtime, when the component is running, if component find that the configuration file does not have a version number, it is considered to be the old configuration file, and the internal convert method will convert the configuration to the corresponding new configuration.

2. keadm provides a conversion command to convert the old configuration file to new configuration. When the component is running, only support the new configuration file.

We use the second option, because:

* There are 3 old configuration files for each component, it is quite different from the new configuration definition and they are eventually discarded in the near future.

* If the component supports the old configuration file, it will add configuration-compatible logic inside the component. We might as well let keadm do this. such as:

```
keadm convertconfig --component=<cloudcore,edgecore,edgesite> --srcdir=<old config dir> --desdir=<new config dir>

```

`srcdir` flag set the dir of the old 2 configfiles.
`desdir` flag set the dir of the new configfile. if `despath` is not set, keadm only print new config, user can create config file by those print info.

keadm first load the old two configfiles and create the new config for each component.

We can gradually abandon this command after the release of several stable versions.


### new config file need version number

Just like kubernetes component config, KubeEdge component config need `apiVersion` to define config version schema .

* cloudcore

```yaml
apiVersion: cloudcore.config.kubeedge.io/v1alpha1
```

* edgecore

```yaml
apiVersion: edgecore.config.kubeedge.io/v1alpha1
```

* edgsite

```yaml
apiVersion: edgesite.config.kubeedge.io/v1alpha1
```

### How to pass the configuration to each module

After the program runs, load the configuration file and use the Register method of each module to pass the configuration to the global variables of each module.


### Use keadm to install and configure KubeEdge components

`keadm` can use the KubeEdge components config api to generate configuration files for each component and allows additional command line flags to override the configuration of each component. This will make it easier to install and configure KubeEdge components.

## Task list tracking

[Task List](https://github.com/kubeedge/kubeedge/issues/1171)
