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
	componentbaseconfig "k8s.io/component-base/config"

	metaconfig "github.com/kubeedge/kubeedge/pkg/apis/componentconfig/meta/v1alpha1"
)

// CloudCoreConfig indicates the config of cloudCore which get from cloudCore config file
type CloudCoreConfig struct {
	metav1.TypeMeta
	// KubeAPIConfig indicates the kubernetes cluster info which cloudCore will connected
	// +Required
	KubeAPIConfig *KubeAPIConfig `json:"kubeAPIConfig,omitempty"`
	// Modules indicates cloudCore modules config
	// +Required
	Modules *Modules `json:"modules,omitempty"`
	// Configuration for LeaderElection
	LeaderElection *componentbaseconfig.LeaderElectionConfiguration `json:"leaderelection,omitempty"`
}

// KubeAPIConfig indicates the configuration for interacting with k8s server
type KubeAPIConfig struct {
	// Master indicates the address of the Kubernetes API server (overrides any value in KubeConfig)
	// such as https://127.0.0.1:8443
	// default ""
	// Note: Can not use "omitempty" option,  It will affect the output of the default configuration file
	Master string `json:"master"`
	// ContentType indicates the ContentType of message transmission when interacting with k8s
	// default "application/vnd.kubernetes.protobuf"
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

// Modules indicates the modules of CloudCore will be use
type Modules struct {
	// CloudHub indicates CloudHub module config
	CloudHub *CloudHub `json:"cloudHub,omitempty"`
	// EdgeController indicates EdgeController module config
	EdgeController *EdgeController `json:"edgeController,omitempty"`
	// DeviceController indicates DeviceController module config
	DeviceController *DeviceController `json:"deviceController,omitempty"`
	// SyncController indicates SyncController module config
	SyncController *SyncController `json:"syncController,omitempty"`
	// CloudStream indicates cloudstream module config
	CloudStream *CloudStream `json:"cloudStream,omitempty"`
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
	// default 1000
	NodeLimit int32 `json:"nodeLimit,omitempty"`
	// TLSCAFile indicates ca file path
	// default "/etc/kubeedge/ca/rootCA.crt"
	TLSCAFile string `json:"tlsCAFile,omitempty"`
	// TLSCAKeyFile indicates caKey file path
	// default "/etc/kubeedge/ca/rootCA.key"
	TLSCAKeyFile string `json:"tlsCAKeyFile,omitempty"`
	// TLSPrivateKeyFile indicates key file path
	// default "/etc/kubeedge/certs/server.crt"
	TLSCertFile string `json:"tlsCertFile,omitempty"`
	// TLSPrivateKeyFile indicates key file path
	// default "/etc/kubeedge/certs/server.key"
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
	// HTTPS indicates https server info
	// +Required
	HTTPS *CloudHubHTTPS `json:"https,omitempty"`
	// AdvertiseAddress sets the IP address for the cloudcore to advertise.
	AdvertiseAddress []string `json:"advertiseAddress,omitempty"`
	// DNSNames sets the DNSNames for CloudCore.
	DNSNames []string `json:"dnsNames,omitempty"`
	// EdgeCertSigningDuration indicates the validity period of edge certificate
	// default 365d
	EdgeCertSigningDuration time.Duration `json:"edgeCertSigningDuration,omitempty"`
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

// CloudHubHttps indicates the http config of CloudHub
type CloudHubHTTPS struct {
	// Enable indicates whether enable Https protocol
	// default true
	Enable bool `json:"enable,omitempty"`
	// Address indicates server ip address
	// default 0.0.0.0
	Address string `json:"address,omitempty"`
	// Port indicates the open port for HTTPS server
	// default 10002
	Port uint32 `json:"port,omitempty"`
}

// EdgeController indicates the config of EdgeController module
type EdgeController struct {
	// Enable indicates whether EdgeController is enabled,
	// if set to false (for debugging etc.), skip checking other EdgeController configs.
	// default true
	Enable bool `json:"enable,omitempty"`
	// NodeUpdateFrequency indicates node update frequency (second)
	// default 10
	NodeUpdateFrequency int32 `json:"nodeUpdateFrequency,omitempty"`
	// Buffer indicates k8s resource buffer
	Buffer *EdgeControllerBuffer `json:"buffer,omitempty"`
	// Context indicates send,receive,response modules for EdgeController module
	Context *EdgeControllerContext `json:"context,omitempty"`
	// Load indicates EdgeController load
	Load *EdgeControllerLoad `json:"load,omitempty"`
}

// EdgeControllerBuffer indicates the EdgeController buffer
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

// EdgeControllerContext indicates the EdgeController context
type EdgeControllerContext struct {
	// SendModule indicates which module will send message to
	SendModule metaconfig.ModuleName `json:"sendModule,omitempty"`
	// ReceiveModule indicates which module will receive message from
	ReceiveModule metaconfig.ModuleName `json:"receiveModule,omitempty"`
	// ResponseModule indicates which module will response message to
	ResponseModule metaconfig.ModuleName `json:"responseModule,omitempty"`
}

// EdgeControllerLoad indicates the EdgeController load
type EdgeControllerLoad struct {
	// UpdatePodStatusWorkers indicates the load of update pod status workers
	// default 1
	UpdatePodStatusWorkers int32 `json:"updatePodStatusWorkers,omitempty"`
	// UpdateNodeStatusWorkers indicates the load of update node status workers
	// default 1
	UpdateNodeStatusWorkers int32 `json:"updateNodeStatusWorkers,omitempty"`
	// QueryConfigMapWorkers indicates the load of query config map workers
	// default 4
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

// SyncController indicates the sync controller
type SyncController struct {
	// Enable indicates whether syncController is enabled,
	// if set to false (for debugging etc.), skip checking other syncController configs.
	// default true
	Enable bool `json:"enable,omitempty"`
}

// CloudSream indicates the stream controller
type CloudStream struct {
	// Enable indicates whether cloudstream is enabled, if set to false (for debugging etc.), skip checking other configs.
	// default true
	Enable bool `json:"enable"`

	// TLSTunnelCAFile indicates ca file path
	// default /etc/kubeedge/ca/rootCA.crt
	TLSTunnelCAFile string `json:"tlsTunnelCAFile,omitempty"`
	// TLSTunnelCertFile indicates cert file path
	// default /etc/kubeedge/certs/server.crt
	TLSTunnelCertFile string `json:"tlsTunnelCertFile,omitempty"`
	// TLSTunnelPrivateKeyFile indicates key file path
	// default /etc/kubeedge/certs/server.key
	TLSTunnelPrivateKeyFile string `json:"tlsTunnelPrivateKeyFile,omitempty"`
	// TunnelPort set open port for tunnel server
	// default 10004
	TunnelPort uint32 `json:"tunnelPort,omitempty"`

	// TLSStreamCAFile indicates kube-apiserver ca file path
	// default /etc/kubeedge/ca/streamCA.crt
	TLSStreamCAFile string `json:"tlsStreamCAFile,omitempty"`
	// TLSStreamCertFile indicates cert file path
	// default /etc/kubeedge/certs/stream.crt
	TLSStreamCertFile string `json:"tlsStreamCertFile,omitempty"`
	// TLSStreamPrivateKeyFile indicates key file path
	// default /etc/kubeedge/certs/stream.key
	TLSStreamPrivateKeyFile string `json:"tlsStreamPrivateKeyFile,omitempty"`
	// StreamPort set open port for stream server
	// default 10003
	StreamPort uint32 `json:"streamPort,omitempty"`
}
