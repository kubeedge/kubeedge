package config

import commonconfig "github.com/kubeedge/kubeedge/pkg/common/apis/config"

const (
	InternalMqttMode = iota // 0: launch an internal mqtt broker.
	BothMqttMode            // 1: launch an internal and external mqtt broker.
	ExternalMqttMode        // 2: launch an external mqtt broker.
)

type EdgeCoreConfig struct {
	Mqtt        *MqttConfig           `json:"mqtt,omitempty"`
	EdgeHub     *EdgeHubConfig        `json:"edgehub,omitempty"`
	Edged       *EdgedConfig          `json:"edged,omitempty"`
	Mesh        *MeshConfig           `json:"mesh,omitempty"`
	Modules     *commonconfig.Modules `json:"modules,omitempty"`
	Metamanager *Metamanager          `json:"metamanager,omitempty"`
}

type MqttConfig struct {
	// external mqtt broker url, default tcp://127.0.0.1:1883
	Server string `json:"server,omitempty"`
	// internal mqtt broker url, default tcp://127.0.0.1:1884
	InternalServer string `json:"internalServer,omitempty"`
	// 0: internal mqtt broker enable only. 1: internal and external mqtt broker enable. 2: external mqtt broker enable only, default: 0
	Mode uint8 `json:"mode,omitempty"`
	// 0: QOSAtMostOnce, 1: QOSAtLeastOnce, 2: QOSExactlyOnce, default 0
	QOS uint8 `json:"qos,omitempty"`
	// if the flag set true, server will store the message and can be delivered to future subscribers, default false
	Retain bool `json:"retain,omitempty"`
	// A size of how many sessions will be handled. default to 100, default 100
	SessionQueueSize int32 `json:"sessionQueueSize,omitempty"`
}

type EdgeHubConfig struct {
	WebSocket  *WebSocketConfig  `json:"webSocket,omitempty"`
	Quic       *QuicConfig       `json:"quic,omitempty"`
	Controller *ControllerConfig `json:"controller,omitempty"`
}

type WebSocketConfig struct {
	// ip:port
	// old : wss://0.0.0.0:10000/e632aba927ea4ac2b575ec1603d56f10/edge-node/events
	Server string `json:"server,omitempty"`
	// TLSCertFile is the file containing x509 Certificate for HTTPS, default /etc/kubeedge/certs/edge.crt
	TLSCertFile string `json:"tlsCertFile,omitempty"`
	// TLSPrivateKeyFile is the file containing x509 private key matching tlsCertFile, default /etc/kubeedge/certs/edge.key
	TLSPrivateKeyFile string `json:"tlsPrivateKeyFile,omitempty"`
	// default  30 #second
	HandshakeTimeout int32 `json:"handshakeTimeout,omitempty"`
	// default 15 # second
	WriteDeadline int32 `json:"writeDeadline,omitempty"`
	// default 15 # second
	ReadDeadline int32 `json:"readDeadline,omitempty"`
}

type QuicConfig struct {
	// ip:port
	Server string `json:"server,omitempty"`
	// default /etc/kubeedge/ca/rootCA.crt
	TLSCaFile string `json:"tlsCaFile,omitempty"`
	// TLSCertFile is the file containing x509 Certificate for HTTPS, default /etc/kubeedge/certs/edge.crt
	TLSCertFile string `json:"tlsCertFile,omitempty"`
	// TLSPrivateKeyFile is the file containing x509 private key matching tlsCertFile, default /etc/kubeedge/certs/edge.key
	TLSPrivateKeyFile string `json:"tlsPrivateKeyFile,omitempty"`
	// default 30 #second
	HandshakeTimeout int32 `json:"handshakeTimeout,omitempty"`
	// default 15 # second
	WriteDeadline int32 `json:"writeDeadline,omitempty"`
	// default 15 # second
	ReadDeadline int32 `json:"readDeadline,omitempty"`
}

type ControllerConfig struct {
	// websocket, quic, default: websocket
	Protocol string `json:"protocol,omitempty"`
	//default 15  # second
	Heartbeat int32 `json:"heartbeat,omitempty"`
	//default e632aba927ea4ac2b575ec1603d56f10
	ProjectId string `json:"projectId,omitempty"`
}

type EdgedConfig struct {
	//default default
	RegisterNodeNamespace string `json:"registerNodeNamespace,omitempty"`
	// default edge-node
	HostnameOverride string `json:"hostnameOverride,omitempty"`
	// default eth0
	InterfaceName string `json:"interfaceName,omitempty"`
	// default 7852396000 #bytes
	EdgedMemoryCapacity int64 `json:"edgedMemoryCapacity,omitempty"`
	// default 10 # second
	NodeStatusUpdateFrequency int32 `json:"nodeStatusUpdateFrequency,omitempty"`
	// default false
	DevicePluginEnabled bool `json:"devicePluginEnabled,omitempty"`
	// default false
	GpuPluginEnabled bool `json:"gpuPluginEnabled,omitempty"`
	//default 80 # percent
	ImageGCHighThreshold int32 `json:"imageGCHighThreshold,omitempty"`
	//default 40 # percent
	ImageGCLowThreshold int32 `json:"imageGCLowThreshold,omitempty"`
	//default 1
	MaximumDeadContainersPerPod int32 `json:"maximumDeadContainersPerPod,omitempty"`
	// default unix:///var/run/docker.sock
	DockerAddress string `json:"dockerAddress,omitempty"`
	// docker, remote, default docker
	RuntimeType string `json:"runtimeType,omitempty"`
	//default unix:///var/run/dockershim.sock
	RemoteRuntimeEndpoint string `json:"remoteRuntimeEndpoint,omitempty"`
	// default unix:///var/run/dockershim.sock
	RemoteImageEndpoint string `json:"remoteImageEndpoint,omitempty"`
	// default 2 #second
	RuntimeRequestTimeout int32 `json:"runtimeRequestTimeout,omitempty"`
	// kubeedge/pause:3.1 for x86 arch
	// kubeedge/pause-arm:3.1 for arm arch
	// kubeedge/pause-arm64 for arm64 arch
	PodsandboxImage string `json:"podsandboxImage,omitempty"`
	//default 60 # second
	ImagePullProgressDeadline int32 `json:"imagePullProgressDeadline,omitempty"`
	// default cgroupfs
	CgroupDriver  string `json:"cgroupDriver,omitempty"`
	NodeIP        string `json:"nodeIP,omitempty"`
	ClusterDNS    string `json:"clusterDNS,omitempty"`
	ClusterDomain string `json:"clusterDomain,omitempty"`
}

type MeshConfig struct {
	Loadbalance *LoadbalanceConfig `json:"loadbalance,omitempty"`
}

type LoadbalanceConfig struct {
	// default RoundRobin
	StrategyName string `json:"strategyName,omitempty"`
}

type Metamanager struct {
	ContextSendGroup  string `json:"contextSendGroup,omitempty"`
	ContextSendModule string `json:"contextSendModule,omitempty"`
	EdgeSite          bool   `json:"edgeSite,omitempty"`
}
