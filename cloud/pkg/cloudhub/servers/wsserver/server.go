package wsserver

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"

	bhLog "github.com/kubeedge/beehive/pkg/common/log"
	"github.com/kubeedge/kubeedge/cloud/pkg/cloudhub/channelq"
	"github.com/kubeedge/kubeedge/cloud/pkg/cloudhub/common/util"
	"github.com/kubeedge/kubeedge/cloud/pkg/cloudhub/handler"

	"github.com/gorilla/mux"
)

// constants for api path
const (
	PathEvent = "/{project_id}/{node_id}/events"
)

// FilterWriter filter writer
type FilterWriter struct{}

func (f *FilterWriter) Write(p []byte) (n int, err error) {
	output := string(p)
	if strings.Contains(output, "http: TLS handshake error from") {
		return 0, nil
	}
	return os.Stderr.Write(p)
}

// StartCloudHub starts the cloud hub service
func StartCloudHub(config *util.Config, eventq *channelq.ChannelEventQueue) error {
	// init certificate
	pool := x509.NewCertPool()
	ok := pool.AppendCertsFromPEM(config.Ca)
	if !ok {
		return fmt.Errorf("fail to load ca content")
	}
	cert, err := tls.X509KeyPair(config.Cert, config.Key)
	if err != nil {
		return err
	}
	tlsConfig := tls.Config{
		ClientCAs:    pool,
		ClientAuth:   tls.RequestClientCert,
		Certificates: []tls.Certificate{cert},
		MinVersion:   tls.VersionTLS12,
		CipherSuites: []uint16{tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256},
	}

	// init handler
	handler.WebSocketHandler = &handler.WebsocketHandle{
		EventHandler: &handler.EventHandle{
			KeepaliveInterval: config.KeepaliveInterval,
			WriteTimeout:      config.WriteTimeout,
			EventQueue:        eventq,
		},
		NodeLimit: config.NodeLimit,
	}
	handler.WebSocketHandler.EventHandler.Handlers = []handler.HandleFunc{handler.WebSocketHandler.EventReadLoop, handler.WebSocketHandler.EventWriteLoop}

	router := mux.NewRouter()
	router.HandleFunc(PathEvent, handler.WebSocketHandler.ServeEvent)

	// start server
	s := http.Server{
		Addr:      fmt.Sprintf("%s:%d", config.Address, config.Port),
		Handler:   router,
		TLSConfig: &tlsConfig,
		ErrorLog:  log.New(&FilterWriter{}, "", log.LstdFlags),
	}
	bhLog.LOGGER.Infof("Start cloud hub websocket server")
	go s.ListenAndServeTLS("", "")

	return nil
}
