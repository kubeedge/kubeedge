package config

import (
	"io/ioutil"
	"path"

	"github.com/kubeedge/kubeedge/common/constants"

	"gopkg.in/yaml.v2"
	"k8s.io/klog"
)

type EdgeCoreConfig struct {
	Mqtt    *MqttConfig    `yaml:"mqtt"`
	EdgeHub *EdgeHubConfig `yaml:"edgehub"`
	Edged   *EdgedConfig   `yaml:"edged"`
	Mesh    *MeshConfig    `yaml:"mesh"`
}

func NewDefaultEdgeCoreConfig() *EdgeCoreConfig {
	return &EdgeCoreConfig{
		Mqtt:    NewDefaultMqttConfig(),
		EdgeHub: NewDefaultEdgeHubConfig(),
		Edged:   NewDefaultEdgedConfig(),
		Mesh:    NewDefaultMeshConfig(),
	}
}

type MqttConfig struct {
	Server           string `yaml:"server"`           //default tcp://127.0.0.1:1883 # external mqtt broker url.
	InternalServer   string `yaml:"internalServer"`   //default tcp://127.0.0.1:1884 # internal mqtt broker url.
	Mode             uint8  `yaml:"mode"`             //default: 0 # 0: internal mqtt broker enable only. 1: internal and external mqtt broker enable. 2: external mqtt broker enable only.
	QOS              uint8  `yaml:"qos"`              //default: 0 # 0: QOSAtMostOnce, 1: QOSAtLeastOnce, 2: QOSExactlyOnce.
	Retain           bool   `yaml:"retain"`           //default: false # if the flag set true, server will store the message and can be delivered to future subscribers.
	SessionQueueSize int32  `yaml:"sessionQueueSize"` //default: 100 # A size of how many sessions will be handled. default to 100.
}

func NewDefaultMqttConfig() *MqttConfig {
	return &MqttConfig{
		Server:           "tcp://127.0.0.1:1883",
		InternalServer:   "tcp://127.0.0.1:1884",
		Mode:             0,
		QOS:              0,
		Retain:           false,
		SessionQueueSize: 100,
	}
}

type EdgeHubConfig struct {
	WebSocket  *WebSocketConfig  `yaml:"websocket"`
	Quic       *QuicConfig       `yaml:"quic"`
	Controller *ControllerConfig `yaml:"controller"`
}

func NewDefaultEdgeHubConfig() *EdgeHubConfig {
	return &EdgeHubConfig{
		WebSocket:  NewDefaultWebSocketConfig(),
		Quic:       NewDefaultQuicConfig(),
		Controller: NewDefaultControllerConfig(),
	}
}

type WebSocketConfig struct {
	Server            string `yaml:"server"`            //new only ip:port , old : wss://0.0.0.0:10000/e632aba927ea4ac2b575ec1603d56f10/edge-node/events
	TLSCertFile       string `yaml:"tlsCertFile"`       //default : /etc/kubeedge/certs/edge.crt
	TLSPrivateKeyFile string `yaml:"tlsPrivateKeyFile"` //default : /etc/kubeedge/certs/edge.key
	HandshakeTimeout  int32  `yaml:"handshakeTimeout"`  //default : 30 #second
	WriteDeadline     int32  `yaml:"writeDeadline"`     //default: 15 # second
	ReadDeadline      int32  `yaml:"readDeadline"`      //default: 15 # second
}

func NewDefaultWebSocketConfig() *WebSocketConfig {
	return &WebSocketConfig{
		Server:            "",
		TLSCertFile:       path.Join(constants.DefaultCertDir, "edge.crt"),
		TLSPrivateKeyFile: path.Join(constants.DefaultCertDir, "edge.key"),
		HandshakeTimeout:  30,
		WriteDeadline:     15,
		ReadDeadline:      15,
	}
}

type QuicConfig struct {
	Server            string `yaml:"server"`            //
	TLSCaFile         string `yaml:"tlsCaFile"`         // default: /etc/kubeedge/ca/rootCA.crt
	TLSCertFile       string `yaml:"tlsCertFile"`       // default: /etc/kubeedge/certs/edge.crt
	TLSPrivateKeyFile string `yaml:"tlsPrivateKeyFile"` // default: /etc/kubeedge/certs/edge.key
	HandshakeTimeout  int32  `yaml:"handshakeTimeout"`  // default: 30 #second
	WriteDeadline     int32  `yaml:"writeDeadline"`     // default: 15 # second
	ReadDeadline      int32  `yaml:"readDeadline"`      // default: 15 # second
}

func NewDefaultQuicConfig() *QuicConfig {
	return &QuicConfig{
		Server:            "",
		TLSCaFile:         path.Join(constants.DefaultCADir, "rootCA.crt"),
		TLSCertFile:       path.Join(constants.DefaultCertDir, "edge.crt"),
		TLSPrivateKeyFile: path.Join(constants.DefaultCertDir, "edge.key"),
		HandshakeTimeout:  30,
		WriteDeadline:     15,
		ReadDeadline:      15,
	}
}

type ControllerConfig struct {
	Protocol  string `yaml:"protocol"`  //default: websocket # websocket, quic
	Heartbeat int32  `yaml:"heartbeat"` //default: 15  # second
	ProjectId string `yaml:"projectId"` //default: e632aba927ea4ac2b575ec1603d56f10
	NodeId    string `yaml:"nodeId"`    //default: edge-node
}

func NewDefaultControllerConfig() *ControllerConfig {
	return &ControllerConfig{
		Protocol:  constants.ProtocolWebsocket,
		Heartbeat: 15,
		ProjectId: "e632aba927ea4ac2b575ec1603d56f10",
		NodeId:    "edge-node",
	}
}

type EdgedConfig struct {
	RegisterNodeNamespace string `yaml:"registerNodeNamespace"` //: default
	// Deprecate, use ControllerConfig.NodeID
	// HostnameOverride string // default: edge-node
	InterfaceName                     string `yaml:"interfaceName"`                     // default: eth0
	EdgedMemoryCapacity               int64  `yaml:"edgedMemoryCapacity"`               // default 7852396000 bytes
	NodeStatusUpdateFrequency         int32  `yaml:"nodeStatusUpdateFrequency"`         // default: 10 # second
	DevicePluginEnabled               bool   `yaml:"devicePluginEnabled"`               // default: false
	GpuPluginEnabled                  bool   `yaml:"gpuPluginEnabled"`                  // default: false
	ImageGCHighThreshold              int32  `yaml:"imageGCHighThreshold"`              //default: 80 # percent
	ImageGCLowThreshold               int32  `yaml:"imageGCLowThreshold"`               //default: 40 # percent
	MaximumDeadContainersPerContainer int32  `yaml:"maximumDeadContainersPerContainer"` //default: 1
	DockerAddress                     string `yaml:"dockerAddress"`                     // default: unix:///var/run/docker.sock
	RuntimeType                       string `yaml:"runtimeType"`                       // default: docker
	RemoteRuntimeEndpoint             string `yaml:"remoteRuntimeEndpoint"`             //default: unix:///var/run/dockershim.sock
	RemoteImageEndpoint               string `yaml:"remoteImageEndpoint"`               // default: unix:///var/run/dockershim.sock
	RuntimeRequestRimeout             int32  `yaml:"runtimeRequestRimeout"`             // default: 2
	PodsandboxImage                   string `yaml:"podsandboxImage"`                   //: kubeedge/pause:3.1 # kubeedge/pause:3.1 for x86 arch , kubeedge/pause-arm:3.1 for arm arch, kubeedge/pause-arm64 for arm64 arch
	ImagePullProgressDeadline         int32  `yaml:"imagePullProgressDeadline"`         //: 60 # second
	CgroupDriver                      string `yaml:"cgroupDriver"`                      // default: cgroupfs
	NodeIP                            string `yaml:"nodeIP"`
	ClusterDNS                        string `yaml:"clusterDNS"`
	ClusterDomain                     string `yaml:"clusterDomain"`
}

func NewDefaultEdgedConfig() *EdgedConfig {
	return &EdgedConfig{
		RegisterNodeNamespace:             "",
		InterfaceName:                     "eth0",
		EdgedMemoryCapacity:               7852396000,
		NodeStatusUpdateFrequency:         10,
		DevicePluginEnabled:               false,
		GpuPluginEnabled:                  false,
		ImageGCHighThreshold:              80,
		ImageGCLowThreshold:               40,
		MaximumDeadContainersPerContainer: 1,
		DockerAddress:                     "unix:///var/run/docker.sock",
		RuntimeType:                       constants.RuntimeTypeDocker,
		RemoteRuntimeEndpoint:             "unix:///var/run/dockershim.sock",
		RemoteImageEndpoint:               "unix:///var/run/dockershim.sock",
		RuntimeRequestRimeout:             2,
		PodsandboxImage:                   "kubeedge/pause:3.1",
		ImagePullProgressDeadline:         60,
		CgroupDriver:                      "cgroupfs",
		NodeIP:                            "",
		ClusterDNS:                        "8.8.8.8",
		ClusterDomain:                     "",
	}
}

type MeshConfig struct {
	Loadbalance *LoadbalanceConfig `yaml:"loadbalance"`
}

func NewDefaultMeshConfig() *MeshConfig {
	return &MeshConfig{
		Loadbalance: NewDefaultLoadbalanceConfig(),
	}
}

type LoadbalanceConfig struct {
	StrategyName string `yaml:"strategyName"` // default: RoundRobin
}

func NewDefaultLoadbalanceConfig() *LoadbalanceConfig {
	return &LoadbalanceConfig{
		StrategyName: constants.StrategyRoundRobin,
	}
}

func (c *EdgeCoreConfig) Parse(fname string) error {
	data, err := ioutil.ReadFile(fname)
	if err != nil {
		klog.Errorf("ReadConfig file %s error %v", fname, err)
		return err
	}
	err = yaml.Unmarshal(data, c)
	if err != nil {
		klog.Errorf("Unmarshal file %s data error %v", fname, err)
		return err
	}
	return nil
}
