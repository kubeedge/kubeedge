package quicserver

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"

	bhLog "github.com/kubeedge/beehive/pkg/common/log"
	"github.com/kubeedge/kubeedge/cloud/pkg/cloudhub/channelq"
	"github.com/kubeedge/kubeedge/cloud/pkg/cloudhub/common/util"
	"github.com/kubeedge/kubeedge/cloud/pkg/cloudhub/handler"
	"github.com/kubeedge/viaduct/pkg/api"
	"github.com/kubeedge/viaduct/pkg/mux"
	"github.com/kubeedge/viaduct/pkg/server"
)

// initServerEntries regist handler func
func initServerEntries() {
	mux.Entry(mux.NewPattern("*").Op("*"), handler.QuicHandler.HandleServer)
}

// StartCloudHub starts the cloud hub service
func StartCloudHub(config *util.Config, eventq *channelq.ChannelEventQueue) {
	// init certificate
	pool := x509.NewCertPool()
	ok := pool.AppendCertsFromPEM(config.Ca)
	if !ok {
		panic(fmt.Errorf("fail to load ca content"))
	}
	cert, err := tls.X509KeyPair(config.Cert, config.Key)
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

	// init handler
	handler.QuicHandler = &handler.QuicHandle{
		EventHandler: &handler.EventHandle{
			KeepaliveInterval: config.KeepaliveInterval,
			WriteTimeout:      config.WriteTimeout,
			EventQueue:        eventq,
		},
		NodeLimit: config.NodeLimit,
	}
	handler.QuicHandler.KeepaliveChannel = make(chan struct{}, 1)
	handler.QuicHandler.EventHandler.Handlers = []handler.HandleFunc{handler.QuicHandler.KeepaliveCheckLoop, handler.QuicHandler.EventWriteLoop}

	initServerEntries()

	svc := server.Server{
		Type:       api.ProtocolTypeQuic,
		Addr:       fmt.Sprintf("%s:%d", config.Address, config.QuicPort),
		TLSConfig:  &tlsConfig,
		AutoRoute:  true,
		ConnNotify: handler.QuicHandler.OnRegister,
		ExOpts:     api.QuicServerOption{MaxIncomingStreams: config.MaxIncomingStreams},
	}
	bhLog.LOGGER.Infof("Start cloud hub quic server")
	svc.ListenAndServeTLS("", "")
}
