package handler

import (
	"time"

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
	EventHandler     *EventHandle
	NodeLimit        int
	KeepaliveChannel chan struct{}
}

// HandleServer handle all the request from quic
func (qh *QuicHandle) HandleServer(container *mux.MessageContainer, writer mux.ResponseWriter) {
	nodeID := container.Header.Get("node_id")
	projectID := container.Header.Get("project_id")

	if qh.EventHandler.GetNodeCount() >= qh.NodeLimit {
		bhLog.LOGGER.Errorf("Fail to serve node %s, reach node limit", nodeID)
		return
	}

	if container.Message.GetOperation() == emodel.OpKeepalive {
		bhLog.LOGGER.Infof("Keepalive message received from node: %s", nodeID)
		qh.KeepaliveChannel <- struct{}{}
		return
	}

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

// KeepaliveCheckLoop checks whether the edge node is still alive
func (qh *QuicHandle) KeepaliveCheckLoop(hi hubio.CloudHubIO, info *emodel.HubInfo, stop chan ExitCode) {
	for {
		keepaliveTimer := time.NewTimer(time.Duration(qh.EventHandler.KeepaliveInterval) * time.Second)
		select {
		case <-qh.KeepaliveChannel:
			bhLog.LOGGER.Infof("Node %s is still alive", info.NodeID)
			keepaliveTimer.Stop()
		case <-keepaliveTimer.C:
			bhLog.LOGGER.Warnf("Timeout to receive heart beat from edge node %s for project %s",
				info.NodeID, info.ProjectID)
			stop <- nodeStop
			return
		}
	}
}

// EventWriteLoop loop write cloud msg to edge
func (qh *QuicHandle) EventWriteLoop(hi hubio.CloudHubIO, info *emodel.HubInfo, stop chan ExitCode) {
	qh.EventHandler.eventWriteLoop(hi, info, stop)
}
