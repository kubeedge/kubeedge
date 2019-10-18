package config

import (
	"path"

	"github.com/kubeedge/kubeedge/common/constants"
	commonconfig "github.com/kubeedge/kubeedge/pkg/common/apis/config"
)

func NewDefaultEdgeCoreConfig() *EdgeCoreConfig {
	return &EdgeCoreConfig{
		Mqtt:        NewDefaultMqttConfig(),
		EdgeHub:     NewDefaultEdgeHubConfig(),
		Edged:       NewDefaultEdgedConfig(),
		Mesh:        NewDefaultMeshConfig(),
		Modules:     NewDefaultModules(),
		Metamanager: NewDefaultMetamanager(),
	}
}

func NewDefaultMqttConfig() *MqttConfig {
	return &MqttConfig{
		Server:           "tcp://127.0.0.1:1883",
		InternalServer:   "tcp://127.0.0.1:1884",
		Mode:             ExternalMqttMode,
		QOS:              0,
		Retain:           false,
		SessionQueueSize: 100,
	}
}

func NewDefaultEdgeHubConfig() *EdgeHubConfig {
	return &EdgeHubConfig{
		WebSocket:  NewDefaultWebSocketConfig(),
		Quic:       NewDefaultQuicConfig(),
		Controller: NewDefaultControllerConfig(),
	}
}

func NewDefaultWebSocketConfig() *WebSocketConfig {
	return &WebSocketConfig{
		Server:            "127.0.0.1:10000",
		TLSCertFile:       path.Join(constants.DefaultCertDir, "edge.crt"),
		TLSPrivateKeyFile: path.Join(constants.DefaultCertDir, "edge.key"),
		HandshakeTimeout:  30,
		WriteDeadline:     15,
		ReadDeadline:      15,
	}
}

func NewDefaultQuicConfig() *QuicConfig {
	return &QuicConfig{
		Server:            "127.0.0.1:10001",
		TLSCaFile:         path.Join(constants.DefaultCADir, "rootCA.crt"),
		TLSCertFile:       path.Join(constants.DefaultCertDir, "edge.crt"),
		TLSPrivateKeyFile: path.Join(constants.DefaultCertDir, "edge.key"),
		HandshakeTimeout:  30,
		WriteDeadline:     15,
		ReadDeadline:      15,
	}
}

func NewDefaultControllerConfig() *ControllerConfig {
	return &ControllerConfig{
		Protocol:  "websocket",
		Heartbeat: 15,
		ProjectId: "e632aba927ea4ac2b575ec1603d56f10",
	}
}

func NewDefaultEdgedConfig() *EdgedConfig {
	return &EdgedConfig{
		RegisterNodeNamespace:       "default",
		HostnameOverride:            "edge-node",
		InterfaceName:               "eth0",
		EdgedMemoryCapacity:         7852396000,
		NodeStatusUpdateFrequency:   10,
		DevicePluginEnabled:         false,
		GpuPluginEnabled:            false,
		ImageGCHighThreshold:        80,
		ImageGCLowThreshold:         40,
		MaximumDeadContainersPerPod: 1,
		DockerAddress:               "unix:///var/run/docker.sock",
		RuntimeType:                 "docker",
		RemoteRuntimeEndpoint:       "unix:///var/run/dockershim.sock",
		RemoteImageEndpoint:         "unix:///var/run/dockershim.sock",
		RuntimeRequestTimeout:       2,
		PodsandboxImage:             "kubeedge/pause:3.1",
		ImagePullProgressDeadline:   60,
		CgroupDriver:                "cgroupfs",
		NodeIP:                      "127.0.0.1",
		ClusterDNS:                  "8.8.8.8",
		ClusterDomain:               "",
	}
}

func NewDefaultMeshConfig() *MeshConfig {
	return &MeshConfig{
		Loadbalance: NewDefaultLoadbalanceConfig(),
	}
}

func NewDefaultLoadbalanceConfig() *LoadbalanceConfig {
	return &LoadbalanceConfig{
		StrategyName: "RoundRobin",
	}
}

func NewDefaultModules() *commonconfig.Modules {
	return &commonconfig.Modules{
		Enabled: []string{"eventbus", "servicebus", "websocket", "metaManager", "edged", "twin", "dbTest", "edgemesh"},
	}
}

func NewDefaultMetamanager() *Metamanager {
	return &Metamanager{
		ContextSendGroup:  "hub",
		ContextSendModule: "websocket",
		EdgeSite:          false,
	}
}
