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
func StartCloudHub(messageq *channelq.ChannelMessageQueue) {
	handler.InitHandler(messageq)
	// start websocket server
	if hubconfig.Get().WebSocket.Enable {
		go startWebsocketServer()
	}
	// start quic server
	if hubconfig.Get().Quic.Enable {
		go startQuicServer()
	}
}

func createTLSConfig(ca, cert, key []byte) tls.Config {
	// init certificate
	pool := x509.NewCertPool()
	ok := pool.AppendCertsFromPEM(ca)
	if !ok {
		panic(fmt.Errorf("fail to load ca content"))
	}
	certificate, err := tls.X509KeyPair(cert, key)
	if err != nil {
		panic(err)
	}
	return tls.Config{
		ClientCAs:    pool,
		ClientAuth:   tls.RequireAndVerifyClientCert,
		Certificates: []tls.Certificate{certificate},
		MinVersion:   tls.VersionTLS12,
		CipherSuites: []uint16{tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256},
	}
}

func startWebsocketServer() {
	tlsConfig := createTLSConfig(hubconfig.Get().Ca, hubconfig.Get().Cert, hubconfig.Get().Key)
	svc := server.Server{
		Type:       api.ProtocolTypeWS,
		TLSConfig:  &tlsConfig,
		AutoRoute:  true,
		ConnNotify: handler.CloudhubHandler.OnRegister,
		Addr:       fmt.Sprintf("%s:%d", hubconfig.Get().WebSocket.Address, hubconfig.Get().WebSocket.Port),
		ExOpts:     api.WSServerOption{Path: "/"},
	}
	klog.Infof("Startting cloudhub %s server", api.ProtocolTypeWS)
	svc.ListenAndServeTLS("", "")
}

func startQuicServer() {
	tlsConfig := createTLSConfig(hubconfig.Get().Ca, hubconfig.Get().Cert, hubconfig.Get().Key)
	svc := server.Server{
		Type:       api.ProtocolTypeQuic,
		TLSConfig:  &tlsConfig,
		AutoRoute:  true,
		ConnNotify: handler.CloudhubHandler.OnRegister,
		Addr:       fmt.Sprintf("%s:%d", hubconfig.Get().Quic.Address, hubconfig.Get().Quic.Port),
		ExOpts:     api.QuicServerOption{MaxIncomingStreams: int(hubconfig.Get().Quic.MaxIncomingStreams)},
	}

	klog.Infof("Startting cloudhub %s server", api.ProtocolTypeQuic)
	svc.ListenAndServeTLS("", "")
}
