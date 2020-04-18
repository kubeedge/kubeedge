package edgestream

import (
	"crypto/tls"
	"net/http"
	"net/url"
	"time"

	"github.com/gorilla/websocket"
	"k8s.io/klog"

	"github.com/kubeedge/beehive/pkg/core"
	"github.com/kubeedge/kubeedge/edge/pkg/edgestream/config"
	"github.com/kubeedge/kubeedge/pkg/apis/componentconfig/edgecore/v1alpha1"
)

//define edgestream module name
const (
	ModuleNameEdgeStream = "edgestream"
	GroupNameEdgeStream  = "edgestream"
)

type edgestream struct {
	enable  bool
	hostkey string
}

func newEdgeStream(enable bool, hostkey string) *edgestream {
	return &edgestream{
		enable:  enable,
		hostkey: hostkey,
	}
}

// Register register edgestream
func Register(s *v1alpha1.EdgeStream, hostkey string) {
	config.InitConfigure(s)
	core.Register(newEdgeStream(s.Enable, hostkey))
}

func (e *edgestream) Name() string {
	return ModuleNameEdgeStream
}

func (e *edgestream) Group() string {
	return GroupNameEdgeStream
}

func (e *edgestream) Enable() bool {
	return e.enable
}

func (e *edgestream) Start() {

	serverURL := url.URL{
		Scheme: "wss",
		Host:   config.Config.TunnelServer,
		Path:   "/v1/kubeedge/connect",
	}

	cert, err := tls.LoadX509KeyPair(config.Config.TLSTunnelCertFile, config.Config.TLSTunnelPrivateKeyFile)
	if err != nil {
		klog.Fatalf("Failed to load x509 key pair: %v", err)
	}

	tlsConfig := &tls.Config{
		InsecureSkipVerify: true,
		Certificates:       []tls.Certificate{cert},
	}

	for range time.NewTicker(time.Second * 2).C {
		err := e.TLSClientConnect(serverURL, tlsConfig)
		if err != nil {
			klog.Errorf("TLSClientConnect error %v", err)
		}
	}
}

func (e *edgestream) TLSClientConnect(url url.URL, tlsConfig *tls.Config) error {
	klog.Info("start a new tunnel stream connection ...")

	dial := websocket.Dialer{
		TLSClientConfig:  tlsConfig,
		HandshakeTimeout: time.Duration(config.Config.HandshakeTimeout) * time.Second,
	}
	header := http.Header{}
	header.Add("ID", e.hostkey)

	con, _, err := dial.Dial(url.String(), header)
	if err != nil {
		klog.Errorf("dial %v error %v", url.String(), err)
		return err
	}
	session := NewTunnelSession(con)
	return session.Serve()
}
