package edgestream

import (
	"crypto/tls"
	"crypto/x509"
	"io/ioutil"
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
		Host:   "127.0.0.1:10250",
		//Host: "localhost:10250",
		Path: "/connect",
	}

	pool := x509.NewCertPool()
	cadate, err := ioutil.ReadFile(config.Config.TLSTunnelCAFile)
	if err != nil {
		klog.Fatalf("read ca file error %v", err)
		return
	}

	pool.AppendCertsFromPEM(cadate)
	tlsConfig := &tls.Config{
		InsecureSkipVerify: false,
		RootCAs:            pool,
	}

	for range time.NewTicker(time.Second).C {
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
		klog.Errorf("dial error %v", err)
		return err
	}
	session := NewTunnelSession(con)
	return session.Serve()
}
