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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	metaconfig "github.com/kubeedge/kubeedge/pkg/apis/meta/v1alpha1"
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
	// Master indicates the address of the Kubernetes API server (overrides any value in Kubeconfig)
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
	// Kubeconfig indicates the path to kubeconfig file with authorization and master location information.
	// default "/root/.kube/config"
	// +Required
	KubeConfig string `json:"kubeConfig,omitempty"`
}

// Modules indicates the modules of cloudcore will be use
type Modules struct {
	// CloudHub indicates cloudhub module config
	CloudHub *CloudHub `json:"cloudhub,omitempty"`
	// EdgeController indicates edgecontroller module config
	EdgeController *EdgeController `json:"edgecontroller,omitempty"`
	// DeviceController indicates devicecontroller module config
	DeviceController *DeviceController `json:"devicecontroller,omitempty"`
}

// CloudHub indicates the config of cloudhub module.
// CloudHub is a web socket or quic server responsible for watching changes at the cloud side, caching and sending messages to EdgeHub.
type CloudHub struct {
	// Enable indicates whether cloudhub is enabled, if set to false (for debugging etc.), skip checking other cloudhub configs.
	// default true
	Enable bool `json:"enable,omitempty"`
	// KeepaliveInterval indicates keep-alive interval (second)
	// default 30
	KeepaliveInterval int32 `json:"keepaliveInterval,omitempty"`
	// NodeLimit indicates node limit
	// default 10
	NodeLimit int32 `json:"nodeLimit,omitempty"`
	// TLSCAFile indicates ca file path
	// default /etc/kubeedge/ca/rootCA.crt
	TLSCAFile string `json:"tlsCAFile,omitempty"`
	// TLSCertFile indicates cert file path
	// default /etc/kubeedge/certs/edge.crt
	TLSCertFile string `json:"tlsCertFile,omitempty"`
	// TLSPrivateKeyFile indicates key file path
	// default /etc/kubeedge/certs/edge.key
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
	// default unix:///var/lib/kubeedge/kubeedge.sock
	Address string `json:"address,omitempty"`
}

// CloudHubWebSocket indicates the websocket config of cloudhub
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

// EdgeController indicates the config of edgecontroller module
type EdgeController struct {
	// Enable indicates whether edgecontroller is enabled, if set to false (for debugging etc.), skip checking other edgecontroller configs.
	// default true
	Enable bool `json:"enable,omitempty"`
	// NodeUpdateFrequency indicates node update frequency (second)
	// default 10
	NodeUpdateFrequency int32 `json:"nodeUpdateFrequency,omitempty"`
	// Buffer indicates k8s resource buffer
	Buffer *EdgeControllerBuffer `json:"buffer,omitempty"`
	// Context indicates send,receive,response modules for edgecontroller module
	Context *EdgeControllerContext `json:"context,omitempty"`
	// Load indicates edgecontroller load
	Load *EdgeControllerLoad `json:"load,omitempty"`
}

// EdgeControllerBuffer indicates the edgecontroller buffer
type EdgeControllerBuffer struct {
	// UpdatePodStatus indicates the buffer of pod status
	// default 1024
	UpdatePodStatus int32 `json:"updatePodStatus,omitempty"`
	// UpdateNodeStatus indicates the buffer of update node status
	// default 1024
	UpdateNodeStatus int32 `json:"updateNodeStatus,omitempty"`
	// QueryConfigmap indicates the buffer of query configmap
	// default 1024
	QueryConfigmap int32 `json:"queryConfigmap,omitempty"`
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
	// ConfigmapEvent indicates the buffer of config map event
	// default 1
	ConfigmapEvent int32 `json:"configmapEvent,omitempty"`
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
	QueryPersistentVolume int32 `json:"queryPersistentvolume,omitempty"`
	// QueryPersistentVolumeClaim indicates the buffer of query persistent volume claim
	// default 1024
	QueryPersistentVolumeClaim int32 `json:"queryPersistentvolumeclaim,omitempty"`
	// QueryVolumeAttachment indicates the buffer of query volume attachment
	// default 1024
	QueryVolumeAttachment int32 `json:"queryVolumeattachment,omitempty"`
	// QueryNode indicates the buffer of query node
	// default 1024
	QueryNode int32 `json:"queryNode,omitempty"`
	// UpdateNode indicates the buffer of update node
	// default 1024
	UpdateNode int32 `json:"updateNode,omitempty"`
}

// EdgeControllerContext indicates the edgecontroller context
type EdgeControllerContext struct {
	// SendModule indicates which module will send message to
	SendModule metaconfig.ModuleName `json:"sendModule,omitempty"`
	// ReceiveModule indicates which module will receive message from
	ReceiveModule metaconfig.ModuleName `json:"receiveModule,omitempty"`
	// ResponseModule indicates which module will response message to
	ResponseModule metaconfig.ModuleName `json:"responseModule,omitempty"`
}

// EdgeControllerLoad indicates the edgecontroller load
type EdgeControllerLoad struct {
	// UpdatePodStatusWorkers indicates the load of update pod status workers
	// default 1
	UpdatePodStatusWorkers int32 `json:"updatePodStatusWorkers,omitempty"`
	// UpdateNodeStatusWorkers indicates the load of update node status workers
	// default 1
	UpdateNodeStatusWorkers int32 `json:"updateNodeStatusWorkers,omitempty"`
	// QueryConfigmapWorkers indicates the load of query config map workers
	// default 1
	QueryConfigmapWorkers int32 `json:"queryConfigmapWorkers,omitempty"`
	// QuerySecretWorkers indicates the load of query secret workers
	// default 4
	QuerySecretWorkers int32 `json:"querySecretWorkers,omitempty"`
	// QueryServiceWorkers indicates the load of query service workers
	// default 4
	QueryServiceWorkers int32 `json:"queryServiceWorkers,omitempty"`
	// QueryEndpointsWorkers indicates the load of query endpointer workers
	// default 4
	QueryEndpointsWorkers int32 `json:"queryEndpointsWorkers,omitempty"`
	// QueryPersistentVolumeWorkers indicates the load of query persistent volume workers
	// default 4
	QueryPersistentVolumeWorkers int32 `json:"queryPersistentVolumeWorkers,omitempty"`
	// QueryPersistentVolumeClaimWorkers indicates the load of query persistent volume claim workers
	// default 4
	QueryPersistentVolumeClaimWorkers int32 `json:"queryPersistentColumeClaimWorkers,omitempty"`
	// QueryVolumeAttachmentWorkers indicates the load of query volume attachment workers
	// default 4
	QueryVolumeAttachmentWorkers int32 `json:"queryVolumeAttachmentWorkers,omitempty"`
	// QueryNodeWorkers indicates the load of query node workers
	// default 4
	QueryNodeWorkers int32 `json:"queryNodeWorkers,omitempty"`
	// UpdateNodeWorkers indicates the load of update node workers
	// default 4
	UpdateNodeWorkers int32 `json:"updateNodeWorkers,omitempty"`
}

// DeviceController indicates the device controller
type DeviceController struct {
	// Enable indicates whether devicecontroller is enabled, if set to false (for debugging etc.), skip checking other devicecontroller configs.
	// default true
	Enable bool `json:"enable,omitempty"`
	// Context indicates send,receive,response modules for devicecontroller module
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

// DeviceControllerBuffer indicates devicecontroller buffer
type DeviceControllerBuffer struct {
	// UpdateDeviceStatus indicates the buffer of update device status
	// default 1024
	UpdateDeviceStatus int32 `json:"updateDeviceStatus,omitempty"`
	// DeviceEvent indicates the buffer of divice event
	// default 1
	DeviceEvent int32 `json:"deviceEvent,omitempty"`
	// DeviceModelEvent indicates the buffer of device model event
	// default 1
	DeviceModelEvent int32 `json:"deviceModelEvent,omitempty"`
}

// DeviceControllerLoad indicates the devicecontroller load
type DeviceControllerLoad struct {
	// UpdateDeviceStatusWorkers indicates the load of update device status workers
	// default 1
	UpdateDeviceStatusWorkers int32 `json:"updateDeviceStatusWorkers,omitempty"`
}
