package servers

import (
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"fmt"

	"k8s.io/klog/v2"

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
	if hubconfig.Config.WebSocket.Enable {
		go startWebsocketServer()
	}
	// start quic server
	if hubconfig.Config.Quic.Enable {
		go startQuicServer()
	}
}

func createTLSConfig(ca, cert, key []byte) tls.Config {
	// init certificate
	pool := x509.NewCertPool()
	ok := pool.AppendCertsFromPEM(pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: ca}))
	if !ok {
		panic(fmt.Errorf("fail to load ca content"))
	}

	certificate, err := tls.X509KeyPair(pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: cert}), pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: key}))
	if err != nil {
		panic(err)
	}
	return tls.Config{
		ClientCAs:    pool,
		ClientAuth:   tls.RequireAndVerifyClientCert,
		Certificates: []tls.Certificate{certificate},
		MinVersion:   tls.VersionTLS12,
		// has to match cipher used by NewPrivateKey method, currently is ECDSA
		CipherSuites: []uint16{tls.TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256},
	}
}

func startWebsocketServer() {
	tlsConfig := createTLSConfig(hubconfig.Config.Ca, hubconfig.Config.Cert, hubconfig.Config.Key)
	svc := server.Server{
		Type:       api.ProtocolTypeWS,
		TLSConfig:  &tlsConfig,
		AutoRoute:  true,
		ConnNotify: handler.CloudhubHandler.OnRegister,
		Addr:       fmt.Sprintf("%s:%d", hubconfig.Config.WebSocket.Address, hubconfig.Config.WebSocket.Port),
		ExOpts:     api.WSServerOption{Path: "/"},
	}
	klog.Infof("Starting cloudhub %s server", api.ProtocolTypeWS)
	klog.Fatal(svc.ListenAndServeTLS("", ""))
}

func startQuicServer() {
	tlsConfig := createTLSConfig(hubconfig.Config.Ca, hubconfig.Config.Cert, hubconfig.Config.Key)
	svc := server.Server{
		Type:       api.ProtocolTypeQuic,
		TLSConfig:  &tlsConfig,
		AutoRoute:  true,
		ConnNotify: handler.CloudhubHandler.OnRegister,
		Addr:       fmt.Sprintf("%s:%d", hubconfig.Config.Quic.Address, hubconfig.Config.Quic.Port),
		ExOpts:     api.QuicServerOption{MaxIncomingStreams: int(hubconfig.Config.Quic.MaxIncomingStreams)},
	}

	klog.Infof("Starting cloudhub %s server", api.ProtocolTypeQuic)
	klog.Fatal(svc.ListenAndServeTLS("", ""))
}
