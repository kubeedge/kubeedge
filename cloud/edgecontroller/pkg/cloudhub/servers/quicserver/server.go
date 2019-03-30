package quicserver

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"

	bhLog "github.com/kubeedge/beehive/pkg/common/log"
	"github.com/kubeedge/kubeedge/cloud/edgecontroller/pkg/cloudhub/channelq"
	hubio "github.com/kubeedge/kubeedge/cloud/edgecontroller/pkg/cloudhub/common/io"
	emodel "github.com/kubeedge/kubeedge/cloud/edgecontroller/pkg/cloudhub/common/model"
	"github.com/kubeedge/kubeedge/cloud/edgecontroller/pkg/cloudhub/common/util"
	"github.com/kubeedge/kubeedge/cloud/edgecontroller/pkg/cloudhub/handler"
	"github.com/kubeedge/viaduct/pkg/api"
	"github.com/kubeedge/viaduct/pkg/conn"
	"github.com/kubeedge/viaduct/pkg/server"
)

//AccessHandle access handler
type AccessHandle struct {
	EventHandle *handler.EventHandle
	NodeLimit   int
}

func (ah *AccessHandle) serveEvent(connection conn.Connection) {
	//state := connection.ConnectionState()

	quicio := &hubio.JsonQuicIO{connection}
	handler.QuicEventHandler.ServeConn(quicio, &emodel.HubInfo{"e632aba927ea4ac2b575ec1603d56f10", "fb4ebb70-2783-42b8-b3ef-63e2fd6d242e"})
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
	ah := &AccessHandle{
		EventHandle: &handler.EventHandle{
			KeepaliveInterval: config.KeepaliveInterval,
			WriteTimeout:      config.WriteTimeout,
			EventQueue:        eventq,
		},
		NodeLimit: config.NodeLimit,
	}
	handler.QuicEventHandler = ah.EventHandle

	svc := server.Server{
		Type:       api.ProtocolTypeQuic,
		Addr:       fmt.Sprintf("%s:%d", config.Address, config.QuicPort),
		TLSConfig:  &tlsConfig,
		AutoRoute:  false,
		ConnNotify: ah.serveEvent,
		ExOpts:     api.QuicServerOption{MaxIncomingStreams: 10000},
	}
	bhLog.LOGGER.Infof("Start cloud hub quic server")
	svc.ListenAndServeTLS("", "")
}
