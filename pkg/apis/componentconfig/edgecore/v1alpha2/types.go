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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	tailoredkubeletconfigv1beta1 "k8s.io/kubelet/config/v1beta1"
	"k8s.io/kubernetes/pkg/apis/core"

	metaconfig "github.com/kubeedge/kubeedge/pkg/apis/componentconfig/meta/v1alpha1"
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
	// FeatureGates is a map of feature names to bools that enable or disable alpha/experimental features.
	FeatureGates map[string]bool `json:"featureGates,omitempty"`
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
	// EdgeStream indicates edgestream module config
	// +Required
	EdgeStream *EdgeStream `json:"edgeStream,omitempty"`
}

// Edged indicates the config fo edged module
// edged is lighted-kubelet
type Edged struct {
	// Enable indicates whether EdgeHub is enabled,
	// if set to false (for debugging etc.), skip checking other EdgeHub configs.
	// default true
	Enable bool `json:"enable"`
	// TailoredKubeletConfig contains the configuration for the Kubelet, tailored by KubeEdge
	TailoredKubeletConfig *tailoredkubeletconfigv1beta1.KubeletConfiguration `json:"tailoredKubeletConfig"`
	// TailoredKubeletFlag
	TailoredKubeletFlag
	// CustomInterfaceName indicates the name of the network interface used for obtaining the IP address.
	// Setting this will override the setting 'NodeIP' if provided.
	// If this is not defined the IP address is obtained by the hostname.
	// default ""
	CustomInterfaceName string `json:"customInterfaceName,omitempty"`
	//RegisterNodeNamespace indicates register node namespace
	// default "default"
	RegisterNodeNamespace string `json:"registerNodeNamespace,omitempty"`
}

type TailoredKubeletFlag struct {
	KubeConfig string `json:"kubeConfig,omitempty"`
	// HostnameOverride is the hostname used to identify the kubelet instead
	// of the actual hostname.
	// default os.Hostname()
	HostnameOverride string `json:"hostnameOverride,omitempty"`
	// NodeIP is IP address of the node.
	// If set, edged will use this IP address for the node.
	NodeIP string `json:"nodeIP,omitempty"`
	// Container-runtime-specific options.
	ContainerRuntimeOptions
	// certDirectory is the directory where the TLS certs are located.
	// If tlsCertFile and tlsPrivateKeyFile are provided, this flag will be ignored.
	CertDirectory string `json:"certDirectory,omitempty"`
	// rootDirectory is the directory path to place kubelet files (volume
	// mounts,etc).
	// default "/var/lib/edged"
	RootDirectory string `json:"rootDirectory,omitempty"`
	// The Kubelet will load its initial configuration from this file.
	// The path may be absolute or relative; relative paths are under the Kubelet's current working directory.
	// Omit this flag to use the combination of built-in default configuration values and flags.
	KubeletConfigFile string `json:"kubeletConfigFile,omitempty"`
	// registerNode enables automatic registration with the apiserver.
	// default true
	// DEPRECATED: This parameter should be set via the TailoredKubeletConfig
	RegisterNode bool `json:"registerNode,omitempty"`
	// registerWithTaints are an array of taints to add to a node object when
	// the edgecore registers itself. This only takes effect when registerNode
	// is true and upon the initial registration of the node.
	RegisterWithTaints []core.Taint `json:"registerWithTaints,omitempty"`
	// WindowsService should be set to true if kubelet is running as a service on Windows.
	// Its corresponding flag only gets registered in Windows builds.
	WindowsService bool `json:"windowsService,omitempty"`
	// WindowsPriorityClass sets the priority class associated with the Kubelet process
	// Its corresponding flag only gets registered in Windows builds
	// The default priority class associated with any process in Windows is NORMAL_PRIORITY_CLASS. Keeping it as is
	// to maintain backwards compatibility.
	// Source: https://docs.microsoft.com/en-us/windows/win32/procthread/scheduling-priorities
	WindowsPriorityClass string `json:"windowsPriorityClass,omitempty"`
	// remoteRuntimeEndpoint is the endpoint of remote runtime service
	// default "unix:///var/run/dockershim.sock"
	RemoteRuntimeEndpoint string `json:"remoteRuntimeEndpoint,omitempty"`
	// remoteImageEndpoint is the endpoint of remote image service
	// default "unix:///var/run/dockershim.sock"
	RemoteImageEndpoint string `json:"remoteImageEndpoint,omitempty"`
	// experimentalMounterPath is the path of mounter binary. Leave empty to use the default mount path
	ExperimentalMounterPath string `json:"experimentalMounterPath,omitempty"`
	// This flag, if set, enables a check prior to mount operations to verify that the required components
	// (binaries, etc.) to mount the volume are available on the underlying node. If the check is enabled
	// and fails the mount operation fails.
	ExperimentalCheckNodeCapabilitiesBeforeMount bool `json:"experimentalCheckNodeCapabilitiesBeforeMount,omitempty"`
	// This flag, if set, will avoid including `EvictionHard` limits while computing Node Allocatable.
	// Refer to [Node Allocatable](https://git.k8s.io/community/contributors/design-proposals/node/node-allocatable.md) doc for more information.
	ExperimentalNodeAllocatableIgnoreEvictionThreshold bool `json:"experimentalNodeAllocatableIgnoreEvictionThreshold,omitempty"`
	// Node Labels are the node labels to add when registering the node in the cluster
	NodeLabels map[string]string `json:"nodeLabels,omitempty"`
	// lockFilePath is the path that kubelet will use to as a lock file.
	// It uses this file as a lock to synchronize with other kubelet processes
	// that may be running.
	LockFilePath string `json:"lockFilePath,omitempty"`
	// ExitOnLockContention is a flag that signifies to the kubelet that it is running
	// in "bootstrap" mode. This requires that 'LockFilePath' has been set.
	// This will cause the kubelet to listen to inotify events on the lock file,
	// releasing it and exiting when another process tries to open that file.
	ExitOnLockContention bool `json:"exitOnLockContention,omitempty"`
	// seccompProfileRoot is the directory path for seccomp profiles.
	SeccompProfileRoot string `json:"seccompProfileRoot,omitempty"`
	// DEPRECATED FLAGS
	// minimumGCAge is the minimum age for a finished container before it is
	// garbage collected.
	MinimumGCAge metav1.Duration `json:"minimumGCAge,omitempty"`
	// maxPerPodContainerCount is the maximum number of old instances to
	// retain per container. Each container takes up some disk space.
	MaxPerPodContainerCount int32 `json:"maxPerPodContainerCount,omitempty"`
	// maxContainerCount is the maximum number of old instances of containers
	// to retain globally. Each container takes up some disk space.
	MaxContainerCount int32 `json:"maxContainerCount,omitempty"`
	// masterServiceNamespace is The namespace from which the kubernetes
	// master services should be injected into pods.
	MasterServiceNamespace string `json:"masterServiceNamespace,omitempty"`
	// registerSchedulable tells the edgecore to register the node as
	// schedulable. Won't have any effect if register-node is false.
	// DEPRECATED: use registerWithTaints instead
	RegisterSchedulable bool `json:"registerSchedulable,omitempty"`
	// nonMasqueradeCIDR configures masquerading: traffic to IPs outside this range will use IP masquerade.
	NonMasqueradeCIDR string `json:"nonMasqueradeCidr,omitempty"`
	// This flag, if set, instructs the edged to keep volumes from terminated pods mounted to the node.
	// This can be useful for debugging volume related issues.
	KeepTerminatedPodVolumes bool `json:"keepTerminatedPodVolumes,omitempty"`
	// SeccompDefault enables the use of `RuntimeDefault` as the default seccomp profile for all workloads on the node.
	// To use this flag, the corresponding SeccompDefault feature gate must be enabled.
	SeccompDefault bool `json:"seccompDefault,omitempty"`
}

// ContainerRuntimeOptions defines options for the container runtime.
type ContainerRuntimeOptions struct {
	// General Options.

	// ContainerRuntime is the container runtime to use.
	// default "docker"
	ContainerRuntime string `json:"containerRuntime,omitempty"`
	// RuntimeCgroups that container runtime is expected to be isolated in.
	RuntimeCgroups string `json:"runtimeCgroups,omitempty"`
	// Docker-specific options.

	// DockershimRootDirectory is the path to the dockershim root directory. Defaults to
	// /var/lib/dockershim if unset. Exposed for integration testing (e.g. in OpenShift).
	DockershimRootDirectory string `json:"dockershimRootDirectory,omitempty"`
	// PodSandboxImage is the image whose network/ipc namespaces
	// containers in each pod will use.
	// default kubeedge/pause:3.1
	PodSandboxImage string `json:"podSandboxImage,omitempty"`
	// DockerEndpoint is the path to the docker endpoint to communicate with.
	DockerEndpoint string `json:"dockerEndpoint,omitempty"`
	// If no pulling progress is made before the deadline imagePullProgressDeadline,
	// the image pulling will be cancelled.
	// Defaults 1m0s.
	// +optional
	ImagePullProgressDeadline metav1.Duration `json:"imagePullProgressDeadline,omitempty"`
	// Network plugin options.

	// networkPluginName is the name of the network plugin to be invoked for
	// various events in kubelet/pod lifecycle
	// default ""
	NetworkPluginName string `json:"networkPluginName,omitempty"`
	// NetworkPluginMTU is the MTU to be passed to the network plugin,
	// and overrides the default MTU for cases where it cannot be automatically
	// computed (such as IPSEC).
	// default 1500
	NetworkPluginMTU int32 `json:"networkPluginMTU,omitempty"`
	// CNIConfDir is the full path of the directory in which to search for
	// CNI config files
	// default "/etc/cni/net.d"
	CNIConfDir string `json:"cniConfDir,omitempty"`
	// CNIBinDir is the full path of the directory in which to search for
	// CNI plugin binaries
	// default "/opt/cni/bin"
	CNIBinDir string `json:"cniBinDir,omitempty"`
	// CNICacheDir is the full path of the directory in which CNI should store
	// cache files
	// default "/var/lib/cni/cache"
	CNICacheDir string `json:"cniCacheDir,omitempty"`

	// Image credential provider plugin options

	// ImageCredentialProviderConfigFile is the path to the credential provider plugin config file.
	// This config file is a specification for what credential providers are enabled and invokved
	// by the kubelet. The plugin config should contain information about what plugin binary
	// to execute and what container images the plugin should be called for.
	// +optional
	ImageCredentialProviderConfigFile string `json:"imageCredentialProviderConfigFile,omitempty"`
	// ImageCredentialProviderBinDir is the path to the directory where credential provider plugin
	// binaries exist. The name of each plugin binary is expected to match the name of the plugin
	// specified in imageCredentialProviderConfigFile.
	// +optional
	ImageCredentialProviderBinDir string `json:"imageCredentialProviderBinDir,omitempty"`
}

// EdgeHub indicates the EdgeHub module config
type EdgeHub struct {
	// Enable indicates whether EdgeHub is enabled,
	// if set to false (for debugging etc.), skip checking other EdgeHub configs.
	// default true
	Enable bool `json:"enable"`
	// Heartbeat indicates heart beat (second)
	// default 15
	Heartbeat int32 `json:"heartbeat,omitempty"`
	// MessageQPS is the QPS to allow while send message to cloudHub.
	// DefaultQPS: 30
	MessageQPS int32 `json:"messageQPS,omitempty"`
	// MessageBurst is the burst to allow while send message to cloudHub.
	// DefaultBurst: 60
	MessageBurst int32 `json:"messageBurst,omitempty"`
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
	// Deprecated: will be removed in future release, will not be saved in configuration file
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
	Enable bool `json:"enable"`
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
	Enable bool `json:"enable"`
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
	Enable bool `json:"enable"`
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
	// MqttSubClientID indicates mqtt subscribe ClientID
	// default ""
	MqttSubClientID string `json:"mqttSubClientID"`
	// MqttPubClientID indicates mqtt publish ClientID
	// default ""
	MqttPubClientID string `json:"mqttPubClientID"`
	// MqttUsername indicates mqtt username
	// default ""
	MqttUsername string `json:"mqttUsername"`
	// MqttPassword indicates mqtt password
	// default ""
	MqttPassword string `json:"mqttPassword"`
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
	Enable bool `json:"enable"`
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
	Enable bool `json:"enable"`
	// ContextSendGroup indicates send group
	ContextSendGroup metaconfig.GroupName `json:"contextSendGroup,omitempty"`
	// ContextSendModule indicates send module
	ContextSendModule metaconfig.ModuleName `json:"contextSendModule,omitempty"`
	// RemoteQueryTimeout indicates remote query timeout (second)
	// default 60
	RemoteQueryTimeout int32 `json:"remoteQueryTimeout,omitempty"`
	// The config of MetaServer
	MetaServer *MetaServer `json:"metaServer,omitempty"`
}

type MetaServer struct {
	Enable            bool   `json:"enable"`
	Server            string `json:"server"`
	TLSCaFile         string `json:"tlsCaFile"`
	TLSCertFile       string `json:"tlsCertFile"`
	TLSPrivateKeyFile string `json:"tlsPrivateKeyFile"`
}

// ServiceBus indicates the ServiceBus module config
type ServiceBus struct {
	// Enable indicates whether ServiceBus is enabled,
	// if set to false (for debugging etc.), skip checking other ServiceBus configs.
	// default false
	Enable bool `json:"enable"`
	// Address indicates address for http server
	Server string `json:"server"`
	// Port indicates port for http server
	Port int `json:"port"`
	// Timeout indicates timeout for servicebus receive mseeage
	Timeout int `json:"timeout"`
}

// DeviceTwin indicates the DeviceTwin module config
type DeviceTwin struct {
	// Enable indicates whether DeviceTwin is enabled,
	// if set to false (for debugging etc.), skip checking other DeviceTwin configs.
	// default true
	Enable bool `json:"enable"`
}

// DBTest indicates the DBTest module config
type DBTest struct {
	// Enable indicates whether DBTest is enabled,
	// if set to false (for debugging etc.), skip checking other DBTest configs.
	// default false
	Enable bool `json:"enable"`
}

// EdgeStream indicates the stream controller
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
