/*
Copyright 2019 The Kubeedge Authors.

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

package common

import (
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/kubeedge/kubeedge/common/constants"
)

// GlobalOptions is global options for edge and init
type GlobalOptions struct {
	KubeEdgeVersion   string
	KubernetesVersion string
	DockerVersion     string
	KubeConfig        string
	LogLevel          string
}

//InitOptions has the kubeedge cloud init information filled by CLI
type InitOptions struct {
	GlobalOptions
	KubeAPIServer  string
	ControllerYaml *ControllerYaml
	LogYaml        *LoggingYaml
	ModulesYaml    *ModulesYaml
}

//JoinOptions has the kubeedge cloud init information filled by CLI
type JoinOptions struct {
	GlobalOptions
	CertPath           string
	EdgeControllerIP   string
	K8SAPIServerIPPort string
	EdgeNodeID         string
	RuntimeType        string

	WebSocketPort uint16
	QuicPort      uint16
	EdgeProjectID string

	EdgeYaml    *EdgeYamlSt
	LogYaml     *LoggingYaml
	ModulesYaml *ModulesYaml
}

//InstallState enum set used for verifying a tool version is installed in host
type InstallState uint8

//Difference enum values for type InstallState
const (
	NewInstallRequired InstallState = iota
	AlreadySameVersionExist
	DefVerInstallRequired
	VersionNAInRepo
)

//ModuleRunning is defined to know the running status of KubeEdge components
type ModuleRunning uint8

//Different possible values for ModuleRunning type
const (
	NoneRunning ModuleRunning = iota
	KubeEdgeCloudRunning
	KubeEdgeEdgeRunning
)

//ToolsInstaller interface for tools with install and teardown methods.
type ToolsInstaller interface {
	InstallTools() error
	TearDown() error
}

//OSTypeInstaller interface for methods to be executed over a specified OS distribution type
type OSTypeInstaller interface {
	IsToolVerInRepo(string, string) (bool, error)
	IsDockerInstalled(string) (InstallState, error)
	InstallDocker() error
	InstallMQTT() error
	IsK8SComponentInstalled(string, string) (InstallState, error)
	InstallK8S() error
	StartK8Scluster() error
	InstallKubeEdge() error
	SetDockerVersion(string)
	SetK8SVersionAndIsNodeFlag(version string, flag bool)
	SetKubeEdgeVersion(string)
	RunEdgeCore() error
	KillKubeEdgeBinary(string) error
	IsKubeEdgeProcessRunning(string) (bool, error)
}

//FlagData stores value and default value of the flags used in this command
type FlagData struct {
	Val    interface{}
	DefVal interface{}
}

//NodeMetaDataLabels defines
type NodeMetaDataLabels struct {
	Name string
}

//NodeMetaDataSt defines
type NodeMetaDataSt struct {
	Name   string
	Labels NodeMetaDataLabels
}

//NodeDefinition defines
type NodeDefinition struct {
	Kind       string
	APIVersion string
	MetaData   NodeMetaDataSt
}

//KubeEdgeControllerConfig has all the below fields; (data taken from "github.com/kubeedge/kubeedge/cloud/pkg/controller/config/kube.go")
type KubeEdgeControllerConfig struct {
	//Master is the url of edge master(kube api server)
	Master string `yaml:"master"`
	//Namespace is the namespace to watch(default is NamespaceAll)
	Namespace string `yaml:"namespace"`
	//ContentType is the content type communicate with edge master(default is "application/vnd.kubernetes.protobuf")
	ContentType string `yaml:"content_type"`
	//QPS is the QPS communicate with edge master(default is 100.0)
	QPS uint `yaml:"qps"`
	//Burst default is 10
	Burst uint `yaml:"burst"`
	//NodeFrequency is the time duration for update node status(default is 20s)
	NodeUpdateFrequency time.Duration `yaml:"node_update_frequency"`
	//KubeConfig is the config used connect to edge master
	KubeConfig string `yaml:"kubeconfig"`
}

//CloudControllerSt consists information to access api-server @ master
type CloudControllerSt struct {
	Kube KubeEdgeControllerConfig `yaml:"kube"`
}

//CloudHubSt represents configuration options for http access
type CloudHubSt struct {
	IPAddress         string `yaml:"address"`
	Port              uint16 `yaml:"port"`
	QuicPort          uint16 `yaml:"quic_port"`
	CA                string `yaml:"ca"`
	Cert              string `yaml:"cert"`
	Key               string `yaml:"key"`
	KeepAliveInterval uint32 `yaml:"keepalive-interval"`
	WriteTimeout      uint32 `yaml:"write-timeout"`
	NodeLimit         uint32 `yaml:"node-limit"`
}

//DeviceControllerSt consists information to access  api-server @ master for Device CRD
type DeviceControllerSt struct {
	Kube KubeEdgeControllerConfig `yaml:"kube"`
}

//ControllerYaml has the edgecontroller yaml configuration/content which shall be written in conf/controller.yaml for cloud component
type ControllerYaml struct {
	Controller       CloudControllerSt  `yaml:"controller"`
	CloudHub         CloudHubSt         `yaml:"cloudhub"`
	DeviceController DeviceControllerSt `yaml:"devicecontroller"`
}

// NewControllerYaml return ControllerYaml instance with default values
func NewControllerYaml() *ControllerYaml {
	return &ControllerYaml{
		Controller: CloudControllerSt{
			Kube: KubeEdgeControllerConfig{
				Master:              DefaultKubeAPIServer,
				Namespace:           constants.DefaultKubeNamespace,
				ContentType:         constants.DefaultKubeContentType,
				QPS:                 constants.DefaultKubeQPS,
				Burst:               constants.DefaultKubeBurst,
				NodeUpdateFrequency: constants.DefaultKubeUpdateNodeFrequency * time.Second,
			},
		},
		CloudHub: CloudHubSt{
			IPAddress:         "0.0.0.0",
			Port:              DefaultWebSocketPort,
			QuicPort:          DefaultQuicPort,
			CA:                "/etc/kubeedge/ca/rootCA.crt",
			Cert:              "/etc/kubeedge/certs/edge.crt",
			Key:               "/etc/kubeedge/certs/edge.key",
			KeepAliveInterval: DefaultKeepAliveInterval,
			WriteTimeout:      DefaultWriteTimeout,
			NodeLimit:         DefaultNodeLimit},
		DeviceController: DeviceControllerSt{
			Kube: KubeEdgeControllerConfig{
				Master:      "http://localhost:8080",
				Namespace:   constants.DefaultKubeNamespace,
				ContentType: constants.DefaultKubeContentType,
				QPS:         constants.DefaultKubeQPS,
				Burst:       constants.DefaultKubeBurst,
				KubeConfig:  ""},
		},
	}
}

//ModulesSt contains the list of modules which shall be added to edgecontroller and edge_core respectively during init
type ModulesSt struct {
	Enabled []string `yaml:"enabled"`
}

//ModulesYaml is the module list which shall be written in conf/modules.yaml for cloud and edge component
type ModulesYaml struct {
	Modules ModulesSt `yaml:"modules"`
}

//LoggingYaml shall be written in conf/logging.yaml for cloud and edge component
type LoggingYaml struct {
	LoggerLevel   string   `yaml:"loggerLevel,omitempty"`
	EnableRsysLog bool     `yaml:"enableRsyslog,omitempty"`
	LogFormatText bool     `yaml:"logFormatText,omitempty"`
	Writers       []string `yaml:"writers,omitempty"`
	LoggerFile    string   `yaml:"loggerFile,omitempty"`
}

//MQTTMode = # 0: internal mqtt broker enable only. 1: internal and external mqtt broker enable. 2: external mqtt broker enable only.
type MQTTMode uint8

//Different message exchange mode supported in KubeEdge using MQTT
const (
	MQTTInternalMode MQTTMode = iota
	MQTTInternalExternalMode
	MQTTExternalMode
)

//MQTTQoSType = # 0: QOSAtMostOnce, 1: QOSAtLeastOnce, 2: QOSExactlyOnce.
type MQTTQoSType uint8

//Different MQTT QoS
const (
	MQTTQoSAtMostOnce MQTTQoSType = iota
	MQTTQoSAtLeastOnce
	MQTTQoSExactlyOnce
)

//MQTTConfig contains MQTT specific config to use MQTT broker
type MQTTConfig struct {
	Server           string      `yaml:"server"`
	InternalServer   string      `yaml:"internal-server"`
	Mode             MQTTMode    `yaml:"mode"`
	QOS              MQTTQoSType `yaml:"qos"`
	Retain           bool        `yaml:"retain"`
	SessionQueueSize uint64      `yaml:"session-queue-size"`
}

//WebSocketSt contains websocket configurations to communicate between CloudHub and EdgeHub
type WebSocketSt struct {
	URL              string `yaml:"url"`
	CertFile         string `yaml:"certfile"`
	KeyFile          string `yaml:"keyfile"`
	HandshakeTimeout uint16 `yaml:"handshake-timeout"`
	WriteDeadline    uint16 `yaml:"write-deadline"`
	ReadDeadline     uint16 `yaml:"read-deadline"`
}

//ControllerSt contain edgecontroller config which edge component uses
type ControllerSt struct {
	Protocol            string `yaml:"protocol"`
	Placement           bool   `yaml:"placement"`
	Heartbeat           uint32 `yaml:"heartbeat"`
	RefreshAKSKInterval uint16 `yaml:"refresh-ak-sk-interval"`
	AuthInfoFilesPath   string `yaml:"auth-info-files-path"`
	PlacementURL        string `yaml:"placement-url"`
	ProjectID           string `yaml:"project-id"`
	NodeID              string `yaml:"node-id"`
}

//EdgeDSt contains configuration required by edged module in KubeEdge component
type EdgeDSt struct {
	RegisterNodeNamespace             string `yaml:"register-node-namespace"`
	HostnameOverride                  string `yaml:"hostname-override"`
	InterfaceName                     string `yaml:"interface-name"`
	NodeStatusUpdateFrequency         uint16 `yaml:"node-status-update-frequency"`
	DevicePluginEnabled               bool   `yaml:"device-plugin-enabled"`
	GPUPluginEnabled                  bool   `yaml:"gpu-plugin-enabled"`
	ImageGCHighThreshold              uint16 `yaml:"image-gc-high-threshold"`
	ImageGCLowThreshold               uint16 `yaml:"image-gc-low-threshold"`
	MaximumDeadContainersPerContainer uint16 `yaml:"maximum-dead-containers-per-container"`
	DockerAddress                     string `yaml:"docker-address"`
	Version                           string `yaml:"version"`
	EdgedMemory                       uint16 `yaml:"edged-memory-capacity-bytes"`
	RuntimeType                       string `yaml:"runtime-type"`
	RuntimeEndpoint                   string `yaml:"remote-runtime-endpoint"`
	ImageEndpoint                     string `yaml:"remote-image-endpoint"`
	RequestTimeout                    uint16 `yaml:"runtime-request-timeout"`
	PodSandboxImage                   string `yaml:"podsandbox-image"`
}
type Mesh struct {
	LB LoadBalance `yaml:"loadbalance"`
}
type LoadBalance struct {
	StrategyName string `yaml:"strategy-name"`
}

//EdgeHubSt contains both websocket and controller config
type EdgeHubSt struct {
	WebSocket  WebSocketSt  `yaml:"websocket"`
	Controller ControllerSt `yaml:"controller"`
	Quic       QuicSt       `yaml:"quic"`
}

type QuicSt struct {
	URL string `yaml:"url"`
}

//EdgeYamlSt content is written into conf/edge.yaml
type EdgeYamlSt struct {
	MQTT    MQTTConfig `yaml:"mqtt"`
	EdgeHub EdgeHubSt  `yaml:"edgehub"`
	EdgeD   EdgeDSt    `yaml:"edged"`
	Mesh    Mesh       `yaml:"mesh"`
}

func NewEdgeYamlSt() *EdgeYamlSt {
	edgeid := uuid.New().String()
	version := "2.0.0"
	url := CreateWebSocketURL("0.0.0.0", uint16(DefaultWebSocketPort), DefaultProjectID, "fb4ebb70-2783-42b8-b3ef-63e2fd6d242e/")

	return &EdgeYamlSt{
		MQTT: MQTTConfig{
			Server:         "tcp://127.0.0.1:1883",
			InternalServer: "tcp://127.0.0.1:1884",
			Mode:           MQTTInternalMode,
			QOS:            MQTTQoSAtMostOnce,
			Retain:         false, SessionQueueSize: 100},
		EdgeHub: EdgeHubSt{
			WebSocket: WebSocketSt{
				URL:              url,
				CertFile:         "/etc/kubeedge/certs/edge.crt",
				KeyFile:          "/etc/kubeedge/certs/edge.key",
				HandshakeTimeout: 30,
				WriteDeadline:    15,
				ReadDeadline:     15},
			Quic: QuicSt{
				URL: "127.0.0.1:10001",
			},
			Controller: ControllerSt{
				Protocol:            DefaultProtocol,
				Placement:           false,
				Heartbeat:           15,
				RefreshAKSKInterval: 10,
				AuthInfoFilesPath:   "/var/IEF/secret",
				PlacementURL:        "https://10.154.193.32:7444/v1/placement_external/message_queue",
				ProjectID:           DefaultProjectID,
				NodeID:              edgeid,
			},
		},
		EdgeD: EdgeDSt{
			RegisterNodeNamespace:             "default",
			HostnameOverride:                  edgeid,
			InterfaceName:                     "eth0",
			NodeStatusUpdateFrequency:         10,
			DevicePluginEnabled:               false,
			GPUPluginEnabled:                  false,
			ImageGCHighThreshold:              80,
			ImageGCLowThreshold:               40,
			MaximumDeadContainersPerContainer: 1,
			DockerAddress:                     "unix:///var/run/docker.sock",
			Version:                           version,
			RuntimeType:                       DefaultRuntimeType,
			RuntimeEndpoint:                   "/var/run/containerd/containerd.sock",
			ImageEndpoint:                     "/var/run/containerd/containerd.sock",
			RequestTimeout:                    2,
			PodSandboxImage:                   "k8s.gcr.io/pause"},
		Mesh: Mesh{
			LB: LoadBalance{
				StrategyName: "RoundRobin",
			},
		},
	}
}

// CreateWebSocketURL create web socket url
func CreateWebSocketURL(address string, port uint16, projectid, id string) string {
	return fmt.Sprintf("wss://%s:%v/%s/%s/events", address, port, projectid, id)
}
