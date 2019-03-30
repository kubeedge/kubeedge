package wsserver

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"

	bhLog "github.com/kubeedge/beehive/pkg/common/log"
	"github.com/kubeedge/kubeedge/cloud/edgecontroller/pkg/cloudhub/channelq"
	hubio "github.com/kubeedge/kubeedge/cloud/edgecontroller/pkg/cloudhub/common/io"
	emodel "github.com/kubeedge/kubeedge/cloud/edgecontroller/pkg/cloudhub/common/model"
	"github.com/kubeedge/kubeedge/cloud/edgecontroller/pkg/cloudhub/common/util"
	"github.com/kubeedge/kubeedge/cloud/edgecontroller/pkg/cloudhub/handler"

	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
)

// constants for api path
const (
	PathEvent = "/{project_id}/{node_id}/events"
)

//AccessHandle access handler
type AccessHandle struct {
	EventHandle *handler.EventHandle
	NodeLimit   int
}

// ServeEvent handle the event coming from websocket
func (ah *AccessHandle) ServeEvent(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	projectID := vars["project_id"]
	nodeID := vars["node_id"]

	if ah.EventHandle.GetNodeCount() >= ah.NodeLimit {
		bhLog.LOGGER.Errorf("fail to serve node %s, reach node limit", nodeID)
		http.Error(w, "too many Nodes connected", http.StatusTooManyRequests)
		return
	}

	upgrader := websocket.Upgrader{}
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		bhLog.LOGGER.Errorf("fail to build websocket connection for node %s, reason %s", nodeID, err.Error())
		http.Error(w, "failed to upgrade to websocket protocol", http.StatusInternalServerError)
		return
	}
	info := &emodel.HubInfo{ProjectID: projectID, NodeID: nodeID}
	hi := &hubio.JSONWSIO{conn}
	ah.EventHandle.ServeConn(hi, info)
}

// ServeQueueWorkload handle workload from queue
func (ah *AccessHandle) ServeQueueWorkload(w http.ResponseWriter, r *http.Request) {
	workload, err := ah.EventHandle.GetWorkload()
	if err != nil {
		bhLog.LOGGER.Errorf("%s", err.Error())
		http.Error(w, "fail to get event queue workload", http.StatusInternalServerError)
		return
	}
	_, err = io.WriteString(w, fmt.Sprintf("%f", workload))
	if err != nil {
		bhLog.LOGGER.Errorf("fail to write string, reason: %s", err.Error())
	}
}

// returns if the event queue is available or not.
// returns 0 if not available and 1 if available.
func (ah *AccessHandle) getEventQueueAvailability() int {
	_, err := ah.EventHandle.GetWorkload()
	if err != nil {
		bhLog.LOGGER.Errorf("eventq is not available, reason %s", err.Error())
		return 0
	}
	return 1
}

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
		ClientAuth:   tls.RequestClientCert,
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
	handler.WebSocketEventHandler = ah.EventHandle

	router := mux.NewRouter()
	router.HandleFunc(PathEvent, ah.ServeEvent)

	// start server
	s := http.Server{
		Addr:      fmt.Sprintf("%s:%d", config.Address, config.Port),
		Handler:   router,
		TLSConfig: &tlsConfig,
		ErrorLog:  log.New(&FilterWriter{}, "", log.LstdFlags),
	}
	bhLog.LOGGER.Infof("Start cloud hub websocket server")
	go s.ListenAndServeTLS("", "")
}
