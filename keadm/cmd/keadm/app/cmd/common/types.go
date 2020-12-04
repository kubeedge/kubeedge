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

package common

import (
	"time"

	"github.com/blang/semver"
)

//InitOptions has the kubeedge cloud init information filled by CLI
type InitOptions struct {
	KubeEdgeVersion  string
	KubeConfig       string
	Master           string
	AdvertiseAddress string
	DNS              string
	TarballPath      string
}

//JoinOptions has the kubeedge cloud init information filled by CLI
type JoinOptions struct {
	InitOptions
	CertPath              string
	CloudCoreIPPort       string
	EdgeNodeName          string
	RuntimeType           string
	RemoteRuntimeEndpoint string
	Token                 string
	CertPort              string
	CGroupDriver          string
}

type CheckOptions struct {
	Domain         string
	DNSIP          string
	IP             string
	Runtime        string
	Timeout        int
	CloudHubServer string
	EdgecoreServer string
	Config         string
}

type CheckObject struct {
	Use  string
	Desc string
	Cmd  string
}

// CollectOptions has the kubeedge debug collect information filled by CLI
type CollectOptions struct {
	Config     string
	OutputPath string
	Detail     bool
	LogPath    string
}

type ResetOptions struct {
	Kubeconfig string
	Force      bool
}

type GettokenOptions struct {
	Kubeconfig string
}

type DiagnoseOptions struct {
	Pod          string
	Namespace    string
	Config       string
	CheckOptions *CheckOptions
	DBPath       string
}

type DiagnoseObject struct {
	Desc string
	Use  string
}

//InstallState enum set used for verifying a tool version is installed in host
type InstallState uint8

//Difference enum values for type InstallState
const (
	NewInstallRequired InstallState = iota
	AlreadySameVersionExist
	ExitError
)

//ModuleRunning is defined to know the running status of KubeEdge components
type ModuleRunning uint8

//Different possible values for ModuleRunning type
const (
	NoneRunning ModuleRunning = iota
	KubeEdgeCloudRunning
	KubeEdgeEdgeRunning
)

//ModuleRunning is defined to know the running status of KubeEdge components
type ComponentType string

//All Component type
const (
	CloudCore ComponentType = "cloudcore"
	EdgeCore  ComponentType = "edgecore"
)

// InstallOptions is defined to know the options for installing kubeedge
type InstallOptions struct {
	ComponentType ComponentType
	TarballPath   string
}

//ToolsInstaller interface for tools with install and teardown methods.
type ToolsInstaller interface {
	InstallTools() error
	TearDown() error
}

//OSTypeInstaller interface for methods to be executed over a specified OS distribution type
type OSTypeInstaller interface {
	InstallMQTT() error
	IsK8SComponentInstalled(string, string) error
	SetKubeEdgeVersion(version semver.Version)
	InstallKubeEdge(InstallOptions) error
	RunEdgeCore() error
	KillKubeEdgeBinary(string) error
	IsKubeEdgeProcessRunning(string) (bool, error)
	IsProcessRunning(string) (bool, error)
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

//ControllerKubeConfig has all the below fields; (data taken from "github.com/kubeedge/kubeedge/cloud/pkg/edgecontroller/config/kube.go")
type ControllerKubeConfig struct {
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

//EdgeControllerSt consists information to access api-server @ master
type EdgeControllerSt struct {
	Kube ControllerKubeConfig `yaml:"kube"`
}

//CloudHubSt represents configuration options for http access
type CloudHubSt struct {
	IPAddress         string `yaml:"address"`
	Port              uint16 `yaml:"port"`
	CA                string `yaml:"ca"`
	Cert              string `yaml:"cert"`
	Key               string `yaml:"key"`
	KeepAliveInterval uint32 `yaml:"keepalive-interval"`
	WriteTimeout      uint32 `yaml:"write-timeout"`
	NodeLimit         uint32 `yaml:"node-limit"`
}

//DeviceControllerSt consists information to access  api-server @ master for Device CRD
type DeviceControllerSt struct {
	Kube ControllerKubeConfig `yaml:"kube"`
}

//CloudCoreYaml has the edgecontroller yaml configuration/content which shall be written in conf/controller.yaml for cloud component
type CloudCoreYaml struct {
	EdgeController   EdgeControllerSt   `yaml:"controller"`
	CloudHub         CloudHubSt         `yaml:"cloudhub"`
	DeviceController DeviceControllerSt `yaml:"devicecontroller"`
}

//ModulesSt contains the list of modules which shall be added to cloudcore and edgecore respectively during init
type ModulesSt struct {
	Enabled []string `yaml:"enabled"`
}

//ModulesYaml is the module list which shall be written in conf/modules.yaml for cloud and edge component
type ModulesYaml struct {
	Modules ModulesSt `yaml:"modules"`
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
	Heartbeat uint32 `yaml:"heartbeat"`
	ProjectID string `yaml:"project-id"`
	NodeID    string `yaml:"node-id"`
}

//EdgeDSt contains configuration required by edged module in KubeEdge component
type EdgeDSt struct {
	RegisterNodeNamespace             string `yaml:"register-node-namespace"`
	HostnameOverride                  string `yaml:"hostname-override"`
	NodeStatusUpdateFrequency         uint16 `yaml:"node-status-update-frequency"`
	DevicePluginEnabled               bool   `yaml:"device-plugin-enabled"`
	GPUPluginEnabled                  bool   `yaml:"gpu-plugin-enabled"`
	ImageGCHighThreshold              uint16 `yaml:"image-gc-high-threshold"`
	ImageGCLowThreshold               uint16 `yaml:"image-gc-low-threshold"`
	MaximumDeadContainersPerContainer uint16 `yaml:"maximum-dead-containers-per-container"`
	DockerAddress                     string `yaml:"docker-address"`
	EdgedMemory                       uint16 `yaml:"edged-memory-capacity-bytes"`
	RuntimeType                       string `yaml:"runtime-type"`
	RuntimeEndpoint                   string `yaml:"remote-runtime-endpoint"`
	ImageEndpoint                     string `yaml:"remote-image-endpoint"`
	RequestTimeout                    uint16 `yaml:"runtime-request-timeout"`
	PodSandboxImage                   string `yaml:"podsandbox-image"`
	ConcurrentConsumers               int    `yaml:"concurrent-consumers"`
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
}

//EdgeYamlSt content is written into conf/edge.yaml
type EdgeYamlSt struct {
	MQTT    MQTTConfig `yaml:"mqtt"`
	EdgeHub EdgeHubSt  `yaml:"edgehub"`
	EdgeD   EdgeDSt    `yaml:"edged"`
	Mesh    Mesh       `yaml:"mesh"`
}
