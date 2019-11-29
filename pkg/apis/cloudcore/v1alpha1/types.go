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

import metaconfig "github.com/kubeedge/kubeedge/pkg/apis/meta/v1alpha1"

type CloudCoreConfig struct {
	metaconfig.TypeMeta
	// Kube set the kubernetes cluster info which cloudcore will connected
	// +Required
	Kube KubeConfig `json:"kube,omitempty"`
	// EdgeController set edgecontroller moudule config
	// +Required
	EdgeController EdgeControllerConfig `json:"edgeController,omitempty"`
	// DeviceController set devicecontroller module config
	// +Required
	DeviceController DeviceControllerConfig `json:"deviceController,omitempty"`
	// Cloudhub set cloudhub module config
	// +Required
	Cloudhub CloudHubConfig `json:"cloudHub,omitempty"`
	// Modules set which modules are enables
	// +Required
	Modules metaconfig.Modules `json:"modules,omitempty"`
}

type EdgeControllerConfig struct {
	// NodeUpdateFrequency set node update frequency (second)
	NodeUpdateFrequency int32 `json:"nodeUpdateFrequency,omitempty"`
	// ControllerContext set send,receive,response modules for edgecontroller module
	ControllerContext ControllerContext `json:"Context"`
}

type DeviceControllerConfig struct {
	// ControllerContext set send,receive,response modules for edgecontroller module
	ControllerContext ControllerContext `json:"Context"`
}

type CloudHubConfig struct {
	// WebSocket set websocket server info
	WebSocket CloudHubWebSocket `json:"websocket,omitempty"`
	// Quic set quic server info
	Quic CloudHubQuic `json:"quic,omitempty"`
	// UnixSocket set unixsocket server info
	UnixSocket CloudHubUnixSocket `json:"unixsocket,omitempty"`

	// TLSCaFile set ca file path
	// default /etc/kubeedge/ca/rootCA.crt
	TLSCAFile string `json:"tlsCAFile,omitempty"`
	// TLSCertFile is the file containing x509 Certificate for HTTPS.
	// default /etc/kubeedge/certs/edge.crt
	TLSCertFile string `json:"tlsCertFile,omitempty"`
	// TLSPrivateKeyFile is the file containing x509 private key matching tlsCertFile,
	// default /etc/kubeedge/certs/edge.key
	TLSPrivateKeyFile string `json:"tlsPrivateKeyFile,omitempty"`
	// KeepaliveInterval set keep alive interval (second)
	// default 30
	KeepaliveInterval uint32 `json:"keepaliveInterval,omitempty"`
	// WriteTimeout set timeout (second)
	// default 30
	WriteTimeout uint32 `json:"writeTimeout,omitempty"`
	// NodeLimit set node limit
	// default 10
	NodeLimit uint32 `json:"nodeLimit,omitempty"`
}

type CloudHubWebSocket struct {
	// EnableWebsocket enable websocket protocol
	// default true
	// Note: Can not use "omitempty" option,  It will affect the output of the default configuration file
	EnableWebsocket bool `json:"enableWebsocket"`
	// Address set server ip address
	// default 0.0.0.0
	Address string `json:"address,omitempty"`
	// WebsocketPort set open port for websocket server
	// default 10000
	WebsocketPort uint32 `json:"websocketPort,omitempty"`
}

type CloudHubQuic struct {
	// EnableQuic enable quic protocol
	// default false
	// Note: Can not use "omitempty" option,  It will affect the output of the default configuration file
	EnableQuic bool `json:"enableQuic,omitempty"`
	// Address set server ip address
	// default 0.0.0.0
	Address string `json:"address,omitempty"`
	// QuicPort set open port for quic server
	// default 10001
	QuicPort uint32 `json:"quicPort,omitempty"`
	// MaxIncomingStreams set the max incoming stream for quic server
	// default 10000
	MaxIncomingStreams int32 `json:"maxIncomingStreams,omitempty"`
}

type CloudHubUnixSocket struct {
	// EnableUnixSocket set enable unix domain socket protocol
	// default true
	// Note: Can not use "omitempty" option,  It will affect the output of the default configuration file
	EnableUnixSocket bool `json:"enableUnixSocket"`
	// UnixSocketAddress set unix domain socket address
	// default unix:///var/lib/kubeedge/kubeedge.sock
	UnixSocketAddress string `json:"unixSocketAddress,omitempty"`
}

type ControllerContext struct {
	// SendModule set which module will send message to
	SendModule metaconfig.ModuleName `json:"sendModule,omitempty"`
	// ReceiveModule set which module will receive message from
	ReceiveModule metaconfig.ModuleName `json:"receiveModule,omitempty"`
	// ResponseModule set which module will response message to
	ResponseModule metaconfig.ModuleName `json:"responseModule,omitempty"`
}

type KubeConfig struct {
	// Master set the address of the Kubernetes API server (overrides any value in Kubeconfig)
	// such as https://127.0.0.1:8443
	// default ""
	// Note: Can not use "omitempty" option,  It will affect the output of the default configuration file
	Master string `json:"master"`
	// Kubeconfig set path to kubeconfig file with authorization and master location information.
	// default "/root/.kube/config"
	KubeConfig string `json:"kubeConfig,omitempty"`
}
