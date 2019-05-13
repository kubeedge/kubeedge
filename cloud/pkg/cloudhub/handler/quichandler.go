package handler

import (
	bhLog "github.com/kubeedge/beehive/pkg/common/log"
	hubio "github.com/kubeedge/kubeedge/cloud/pkg/cloudhub/common/io"
	emodel "github.com/kubeedge/kubeedge/cloud/pkg/cloudhub/common/model"
	"github.com/kubeedge/viaduct/pkg/conn"
	"github.com/kubeedge/viaduct/pkg/mux"
)

//QuicHandler handle all request from quic
var QuicHandler *QuicHandle

//QuicHandle access handler
type QuicHandle struct {
	EventHandler *EventHandle
	NodeLimit    int
}

// HandleServer handle all the request from quic
func (qh *QuicHandle) HandleServer(container *mux.MessageContainer, writer mux.ResponseWriter) {
	nodeID := container.Header.Get("node_id")
	projectID := container.Header.Get("project_id")
	err := qh.EventHandler.Pub2Controller(&emodel.HubInfo{ProjectID: projectID, NodeID: nodeID}, container.Message)
	if err != nil {
		// if err, we should stop node, write data to edgehub, stop nodify
		bhLog.LOGGER.Errorf("Failed to serve handle with error: %s", err.Error())
	}
}

// OnRegister regist node on first connection
func (qh *QuicHandle) OnRegister(connection conn.Connection) {
	nodeID := connection.ConnectionState().Headers.Get("node_id")
	projectID := connection.ConnectionState().Headers.Get("project_id")

	quicio := &hubio.JSONQuicIO{Connection: connection}
	go qh.EventHandler.ServeConn(quicio, &emodel.HubInfo{ProjectID: projectID, NodeID: nodeID})
}

// EventWriteLoop loop write cloud msg to edge
func (qh *QuicHandle) EventWriteLoop(hi hubio.CloudHubIO, info *emodel.HubInfo, stop chan ExitCode) {
	qh.EventHandler.eventWriteLoop(hi, info, stop)
}
