package servers

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"

	"k8s.io/klog"

	"github.com/kubeedge/kubeedge/cloud/pkg/cloudhub/channelq"
	hubconfig "github.com/kubeedge/kubeedge/cloud/pkg/cloudhub/config"
	"github.com/kubeedge/kubeedge/cloud/pkg/cloudhub/handler"
	"github.com/kubeedge/viaduct/pkg/api"
	"github.com/kubeedge/viaduct/pkg/server"
)

// StartCloudHub starts the cloud hub service
func StartCloudHub(protocolType string, c *hubconfig.Configure, messageq *channelq.ChannelMessageQueue) {
	// init certificate
	pool := x509.NewCertPool()
	ok := pool.AppendCertsFromPEM(c.Ca)
	if !ok {
		panic(fmt.Errorf("fail to load ca content"))
	}
	cert, err := tls.X509KeyPair(c.Cert, c.Key)
	if err != nil {
		panic(err)
	}
	tlsConfig := tls.Config{
		ClientCAs:    pool,
		ClientAuth:   tls.RequireAndVerifyClientCert,
		Certificates: []tls.Certificate{cert},
		MinVersion:   tls.VersionTLS12,
		CipherSuites: []uint16{tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256},
	}

	handler.InitHandler(c, messageq)

	svc := server.Server{
		Type:       protocolType,
		TLSConfig:  &tlsConfig,
		AutoRoute:  true,
		ConnNotify: handler.CloudhubHandler.OnRegister,
	}

	switch protocolType {
	case api.ProtocolTypeWS:
		svc.Addr = fmt.Sprintf("%s:%d", c.WebSocket.Address, c.WebSocket.Port)
		svc.ExOpts = api.WSServerOption{Path: "/"}
	case api.ProtocolTypeQuic:
		svc.Addr = fmt.Sprintf("%s:%d", c.Quic.Address, c.Quic.Port)
		svc.ExOpts = api.QuicServerOption{MaxIncomingStreams: int(c.Quic.MaxIncomingStreams)}
	default:
		panic(fmt.Errorf("invalid protocol, should be websocket or quic"))
	}

	klog.Infof("Start cloud hub %s server", protocolType)
	svc.ListenAndServeTLS("", "")
}
