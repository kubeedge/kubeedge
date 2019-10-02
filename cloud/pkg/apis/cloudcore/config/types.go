package config

import (
	"io/ioutil"
	"path"

	"gopkg.in/yaml.v2"
	"k8s.io/klog"

	"github.com/kubeedge/kubeedge/common/constants"
)

type CloudCoreConfig struct {
	Kube           *KubeConfig           `yaml:"kube"`
	EdgeController *EdgeControllerConfig `yaml:"edgeController"`
	Cloudhub       *CloudHubConfig       `yaml:"cloudHub"`
	Modules        *Modules              `yaml:"modules"`
}

func NewDefaultCloudCoreConfig() *CloudCoreConfig {
	return &CloudCoreConfig{
		Kube:           NewDefaultKubeConfig(),
		EdgeController: NewDefaultEdgeControllerConfig(),
		Cloudhub:       NewDefaultCloudHubConfig(),
		Modules:        NewDefaultModules(),
	}
}

type EdgeControllerConfig struct {
	NodeUpdateFrequency int32 `yaml:"nodeUpdateFrequency"`
}

func NewDefaultEdgeControllerConfig() *EdgeControllerConfig {
	return &EdgeControllerConfig{
		NodeUpdateFrequency: 10,
	}
}

type CloudHubConfig struct {
	EnableWebsocket    bool   `yaml:"enableWebsocket"`    //default true # enable websocket protocol
	WebsocketPort      int32  `yaml:"websocketPort"`      //default 10000 # open port for websocket server
	EnableQuic         bool   `yaml:"enableQuic"`         //default false # enable quic protocol
	QuicPort           int32  `yaml:"quicPort"`           //default 10001 # open prot for quic server
	MaxIncomingStreams int32  `yaml:"maxIncomingStreams"` //default 10000 # the max incoming stream for quic server
	EnableUnixSocket   bool   `yaml:"enableUnixSocket"`   //default true # enable unix domain socket protocol
	UnixSocketAddress  string `yaml:"unixSocketAddress"`  //default unix:///var/lib/kubeedge/kubeedge.sock # unix domain socket address
	Address            string `yaml:"address"`            //default 0.0.0.0
	TLSCaFile          string `yaml:"tlsCaFile"`          //default /etc/kubeedge/ca/rootCA.crt
	TLSCertFile        string `yaml:"tlsCertFile"`        //default /etc/kubeedge/certs/edge.crt
	TLSPrivateKeyFile  string `yaml:"tlsPrivateKeyFile"`  //default /etc/kubeedge/certs/edge.key
	KeepaliveInterval  int32  `yaml:"keepaliveInterval"`  //default 30
	WriteTimeout       int32  `yaml:"writeTimeout"`       //default 30
	NodeLimit          int32  `yaml:"nodeLimit"`          //default 10
}

func NewDefaultCloudHubConfig() *CloudHubConfig {
	return &CloudHubConfig{
		EnableWebsocket:    true,
		WebsocketPort:      10000,
		EnableQuic:         false,
		QuicPort:           10001,
		MaxIncomingStreams: 10000,
		EnableUnixSocket:   true,
		UnixSocketAddress:  "unix:///var/lib/kubeedge/kubeedge.sock",
		Address:            "0.0.0.0",
		TLSCaFile:          path.Join(constants.DefaultCADir, "rootCA.crt"),
		TLSCertFile:        path.Join(constants.DefaultCertDir, "edge.crt"),
		TLSPrivateKeyFile:  path.Join(constants.DefaultCertDir, "edge.key"),
		KeepaliveInterval:  30,
		WriteTimeout:       30,
		NodeLimit:          10,
	}
}

type KubeConfig struct {
	Master     string `yaml:"master"`     // kube-apiserver address (such as:http://localhost:8080)
	Kubeconfig string `yaml:"kubeconfig"` // default "/root/.kube/config"
}

func NewDefaultKubeConfig() *KubeConfig {
	return &KubeConfig{
		Master:     "",
		Kubeconfig: "/root/.kube/config",
	}
}

type Modules struct {
	Enabled []string `yaml:"enabled"` //default devicecontroller, edgecontroller, cloudhub
}

func NewDefaultModules() *Modules {
	return &Modules{
		Enabled: []string{"devicecontroller", "edgecontroller", "cloudhub"},
	}
}

// TODO @kadisi  add AdmissionControllerConfig
type AdmissionControllerConfig struct {
}

func NewDefaultAdmissionControllerConfig() *AdmissionControllerConfig {
	return &AdmissionControllerConfig{}
}

func (c *CloudCoreConfig) Parse(fname string) error {
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
