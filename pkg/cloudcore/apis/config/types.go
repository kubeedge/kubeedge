package config

import commonconfig "github.com/kubeedge/kubeedge/pkg/common/apis/config"

type CloudCoreConfig struct {
	Kube             *KubeConfig             `json:"kube,omitempty"`
	EdgeController   *EdgeControllerConfig   `json:"edgeController,omitempty"`
	DeviceController *DeviceControllerConfig `json:"deviceController,omitempty"`
	Cloudhub         *CloudHubConfig         `json:"cloudHub,omitempty"`
	Modules          *commonconfig.Modules   `json:"modules,omitempty"`
}

type EdgeControllerConfig struct {
	NodeUpdateFrequency int32              `json:"nodeUpdateFrequency,omitempty"`
	ControllerContext   *ControllerContext `json:"Context"`
}

type DeviceControllerConfig struct {
	ControllerContext *ControllerContext `json:"Context"`
}

type CloudHubConfig struct {
	// enable websocket protocol ,default true
	EnableWebsocket bool `json:"enableWebsocket,omitempty"`
	// open port for websocket server, default 10000
	WebsocketPort int32 `json:"websocketPort,omitempty"`
	// enable quic protocol, default false
	EnableQuic bool `json:"enableQuic,omitempty"`
	// open prot for quic server, default 10001
	QuicPort int32 `json:"quicPort,omitempty"`
	// the max incoming stream for quic server, default 10000
	MaxIncomingStreams int32 `json:"maxIncomingStreams,omitempty"`
	// enable unix domain socket protocol, default true
	EnableUnixSocket bool `json:"enableUnixSocket,omitempty"`
	// unix domain socket address, default unix:///var/lib/kubeedge/kubeedge.sock
	UnixSocketAddress string `json:"unixSocketAddress,omitempty"`
	//default 0.0.0.0
	Address string `json:"address,omitempty"`
	//default /etc/kubeedge/ca/rootCA.crt
	TLSCaFile string `json:"tlsCaFile,omitempty"`
	// TLSCertFile is the file containing x509 Certificate for HTTPS.  default /etc/kubeedge/certs/edge.crt
	TLSCertFile string `json:"tlsCertFile,omitempty"`
	// TLSPrivateKeyFile is the file containing x509 private key matching tlsCertFile, default /etc/kubeedge/certs/edge.key
	TLSPrivateKeyFile string `json:"tlsPrivateKeyFile,omitempty"`
	//default 30
	KeepaliveInterval int32 `json:"keepaliveInterval,omitempty"`
	//default 30
	WriteTimeout int32 `json:"writeTimeout,omitempty"`
	//default 10
	NodeLimit int32 `json:"nodeLimit,omitempty"`
}

// TODO @kadisi  add AdmissionControllerConfig
type AdmissionControllerConfig struct {
}

type ControllerContext struct {
	SendModule     string `json:"sendModule,omitempty"`
	ReceiveModule  string `json:"receiveModule,omitempty"`
	ResponseModule string `json:"responseModule,omitempty"`
}

type KubeConfig struct {
	// The address of the Kubernetes API server (overrides any value in kubeconfig)
	Master string `json:"master,omitempty"`
	// Path to kubeconfig file with authorization and master location information. default "/root/.kube/config"
	Kubeconfig string `json:"kubeconfig,omitempty"`
}
