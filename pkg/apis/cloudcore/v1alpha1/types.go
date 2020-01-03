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

type CloudCoreConfig struct {
	metav1.TypeMeta
	// KubeAPIConfig set the kubernetes cluster info which cloudcore will connected
	// +Required
	KubeAPIConfig KubeAPIConfig `json:"kubeAPIConfig,omitempty"`
	// Modules set cloudcore modules config
	// +Required
	Modules Modules `json:"modules,omitempty"`
}

type KubeAPIConfig struct {
	// Master set the address of the Kubernetes API server (overrides any value in Kubeconfig)
	// such as https://127.0.0.1:8443
	// default ""
	// Note: Can not use "omitempty" option,  It will affect the output of the default configuration file
	Master string `json:"master"`
	// ContentType set the ContentType of message transmission when interacting with k8s
	// default application/vnd.kubernetes.protobuf
	ContentType string `json:"contentType,omitempty"`
	// QPS to use while talking with kubernetes apiserve
	// default 100
	QPS int32 `json:"qps,omitempty"`
	// Burst to use while talking with kubernetes apiserver
	// default 200
	Burst int32 `json:"burst,omitempty"`
	// Kubeconfig set path to kubeconfig file with authorization and master location information.
	// default "/root/.kube/config"
	KubeConfig string `json:"kubeConfig,omitempty"`
}

type Modules struct {
	// CloudHub set cloudhub module config
	CloudHub CloudHub `json:"cloudhub"`
	// EdgeController set edgecontroller module config
	EdgeController EdgeController `json:"edgecontroller"`
	// DeviceController set devicecontroller module config
	DeviceController DeviceController `json:"devicecontroller"`
}

type CloudHub struct {
	// Enable set whether use this module, if false , need check other config
	// default true
	Enable bool `json:"enable,omitempty"`
	// KeepaliveInterval set keep alive interval (second)
	// default 30
	KeepaliveInterval int32 `json:"keepaliveInterval,omitempty"`
	// NodeLimit set node limit
	// default 10
	NodeLimit int32 `json:"nodeLimit,omitempty"`
	// TLSCAFile set ca file path
	// default /etc/kubeedge/ca/rootCA.crt
	TLSCAFile string `json:"tlsCAFile,omitempty"`
	// TLSCertFile set cert file path
	// default /etc/kubeedge/certs/edge.crt
	TLSCertFile string `json:"tlsCertFile,omitempty"`
	// TLSPrivateKeyFile set key file path
	// default /etc/kubeedge/certs/edge.key
	TLSPrivateKeyFile string `json:"tlsPrivateKeyFile,omitempty"`
	// WriteTimeout set write time (second)
	// default 30
	WriteTimeout int32 `json:"writeTimeout,omitempty"`
	// Quic set quic server info
	// +Required
	Quic CloudHubQuic `json:"quic,omitempty"`
	// UnixSocket set unixsocket server info
	// +Required
	UnixSocket CloudHubUnixSocket `json:"unixsocket,omitempty"`
	// WebSocket set websocket server info
	// +Required
	WebSocket CloudHubWebSocket `json:"websocket,omitempty"`
}

type CloudHubQuic struct {
	// Enable enable quic protocol
	// default false
	// Note: Can not use "omitempty" option,  It will affect the output of the default configuration file
	Enable bool `json:"enable"`
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

type CloudHubUnixSocket struct {
	// Enable set enable unix domain socket protocol
	// default true
	// Note: Can not use "omitempty" option,  It will affect the output of the default configuration file
	Enable bool `json:"enable"`
	// Address set unix domain socket address
	// default unix:///var/lib/kubeedge/kubeedge.sock
	Address string `json:"address,omitempty"`
}

type CloudHubWebSocket struct {
	// Enable enable websocket protocol
	// default true
	// Note: Can not use "omitempty" option,  It will affect the output of the default configuration file
	Enable bool `json:"enable"`
	// Address set server ip address
	// default 0.0.0.0
	Address string `json:"address,omitempty"`
	// Port set open port for websocket server
	// default 10000
	Port uint32 `json:"port,omitempty"`
}

type EdgeController struct {
	// Enable set whether use this module,if false no need check other config
	// default true
	// Note: Can not use "omitempty" option,  It will affect the output of the default configuration file
	Enable bool `json:"enable"`
	// NodeUpdateFrequency set node update frequency (second)
	// default 10
	NodeUpdateFrequency int32 `json:"nodeUpdateFrequency,omitempty"`
	// Buffer set k8s resource buffer
	Buffer EdgeControllerBuffer `json:"buffer,omitempty"`
	// Context set send,receive,response modules for edgecontroller module
	Context EdgeControllerContext `json:"context,omitempty"`
	// Load set load
	Load EdgeControllerLoad `json:"load,omitempty"`
}

type EdgeControllerBuffer struct {
	// UpdatePodStatus set pod status buffer
	// default 1024
	UpdatePodStatus int32 `json:"updatePodStatus,omitempty"`
	// UpdateNodeStatus set update node status buffer
	// default 1024
	UpdateNodeStatus int32 `json:"updateNodeStatus,omitempty"`
	// QueryConfigmap set query configmap buffer
	// default 1024
	QueryConfigmap int32 `json:"queryConfigmap,omitempty"`
	// QuerySecret set query secret buffer
	// default 1024
	QuerySecret int32 `json:"querySecret,omitempty"`
	// QueryService set query service buffer
	// default 1024
	QueryService int32 `json:"queryService,omitempty"`
	// QueryEndpoints set query endpoint buffer
	// default 1024
	QueryEndpoints int32 `json:"queryEndpoints,omitempty"`
	// PodEvent set pod event buffer
	// default 1
	PodEvent int32 `json:"podEvent,omitempty"`
	// ConfigmapEvent set config map event buffer
	// default 1
	ConfigmapEvent int32 `json:"configmapEvent,omitempty"`
	// SecretEvent set secret event buffer
	// default 1
	SecretEvent int32 `json:"secretEvent,omitempty"`
	// ServiceEvent set service event buffer
	// default 1
	ServiceEvent int32 `json:"serviceEvent,omitempty"`
	// EndpointsEvent set endpoint event
	// default 1
	EndpointsEvent int32 `json:"endpointsEvent,omitempty"`
	// QueryPersistentvolume set query persistent volume buffer
	// default 1024
	QueryPersistentvolume int32 `json:"queryPersistentvolume,omitempty"`
	// QueryPersistentvolumeclaim set query persistent volume claim buffer
	// default 1024
	QueryPersistentvolumeclaim int32 `json:"queryPersistentvolumeclaim,omitempty"`
	// QueryVolumeattachment set query volume attachment buffer
	// default 1024
	QueryVolumeattachment int32 `json:"queryVolumeattachment,omitempty"`
	// QueryNode set query node buffer
	// default 1024
	QueryNode int32 `json:"queryNode,omitempty"`
	// UpdateNode set update node buffer
	// default 1024
	UpdateNode int32 `json:"updateNode,omitempty"`
}

type EdgeControllerContext struct {
	// SendModule set which module will send message to
	SendModule metaconfig.ModuleName `json:"sendModule,omitempty"`
	// ReceiveModule set which module will receive message from
	ReceiveModule metaconfig.ModuleName `json:"receiveModule,omitempty"`
	// ResponseModule set which module will response message to
	ResponseModule metaconfig.ModuleName `json:"responseModule,omitempty"`
}

type EdgeControllerLoad struct {
	// default 1
	UpdatePodStatusWorkers int32 `json:"updatePodStatusWorkers,omitempty"`
	// default 1
	UpdateNodeStatusWorkers int32 `json:"updateNodeStatusWorkers,omitempty"`
	// default 1
	QueryConfigmapWorkers int32 `json:"queryConfigmapWorkers,omitempty"`
	// default 4
	QuerySecretWorkers int32 `json:"querySecretWorkers,omitempty"`
	// default 4
	QueryServiceWorkers int32 `json:"queryServiceWorkers,omitempty"`
	// default 4
	QueryEndpointsWorkers int32 `json:"queryEndpointsWorkers,omitempty"`
	// default 4
	QueryPersistentvolumeWorkers int32 `json:"queryPersistentvolumeWorkers,omitempty"`
	// default 4
	QueryPersistentvolumeclaimWorkers int32 `json:"queryPersistentvolumeclaimWorkers,omitempty"`
	// default 4
	QueryVolumeattachmentWorkers int32 `json:"queryVolumeattachmentWorkers,omitempty"`
	// default 4
	QueryNodeWorkers int32 `json:"queryNodeWorkers,omitempty"`
	// default 4
	UpdateNodeWorkers int32 `json:"updateNodeWorkers,omitempty"`
}

type DeviceController struct {
	// Enable set whether use this module, if false , need check other config
	// default true
	Enable bool `json:"enable,omitempty"`
	// Context set send,receive,response modules for devicecontroller module
	Context DeviceControllerContext `json:"context"`
	// Buffer set Device controller buffer
	Buffer DeviceControllerBuffer `json:"buffer"`
	// Load set DeviceController Load
	Load DeviceControllerLoad `json:"load"`
}
type DeviceControllerContext struct {
	// SendModule set which module will send message to
	SendModule metaconfig.ModuleName `json:"sendModule,omitempty"`
	// ReceiveModule set which module will receive message from
	ReceiveModule metaconfig.ModuleName `json:"receiveModule,omitempty"`
	// ResponseModule set which module will response message to
	ResponseModule metaconfig.ModuleName `json:"responseModule,omitempty"`
}
type DeviceControllerBuffer struct {
	// default 1024
	UpdateDeviceStatus int32 `json:"updateDeviceStatus,omitempty"`
	// default 1
	DeviceEvent int32 `json:"deviceEvent,omitempty"`
	// default 1
	DeviceModelEvent int32 `json:"deviceModelEvent,omitempty"`
}

type DeviceControllerLoad struct {
	// default 1
	UpdateDeviceStatusWorkers int32 `json:"updateDeviceStatusWorkers,omitempty"`
}
