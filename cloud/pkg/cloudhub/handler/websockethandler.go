package handler

import (
	"fmt"
	"io"
	"net/http"
	"time"

	bhLog "github.com/kubeedge/beehive/pkg/common/log"
	"github.com/kubeedge/beehive/pkg/core/model"
	hubio "github.com/kubeedge/kubeedge/cloud/pkg/cloudhub/common/io"
	emodel "github.com/kubeedge/kubeedge/cloud/pkg/cloudhub/common/model"

	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
)

// WebSocketHandler handle all request from websocket
var WebSocketHandler *WebsocketHandle

// WebsocketHandle access handler
type WebsocketHandle struct {
	EventHandler *EventHandle
	NodeLimit    int
}

// ServeEvent handle the event coming from websocket
func (wh *WebsocketHandle) ServeEvent(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	projectID := vars["project_id"]
	nodeID := vars["node_id"]

	if wh.EventHandler.GetNodeCount() >= wh.NodeLimit {
		bhLog.LOGGER.Errorf("Fail to serve node %s, reach node limit", nodeID)
		http.Error(w, "too many Nodes connected", http.StatusTooManyRequests)
		return
	}

	upgrader := websocket.Upgrader{}
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		bhLog.LOGGER.Errorf("Fail to build websocket connection for node %s, reason %s", nodeID, err.Error())
		http.Error(w, "failed to upgrade to websocket protocol", http.StatusInternalServerError)
		return
	}
	info := &emodel.HubInfo{ProjectID: projectID, NodeID: nodeID}
	hi := &hubio.JSONWSIO{WSConn: conn}
	wh.EventHandler.ServeConn(hi, info)
}

// ServeQueueWorkload handle workload from queue
func (wh *WebsocketHandle) ServeQueueWorkload(w http.ResponseWriter, r *http.Request) {
	workload, err := wh.EventHandler.GetWorkload()
	if err != nil {
		bhLog.LOGGER.Errorf("%s", err.Error())
		http.Error(w, "Fail to get event queue workload", http.StatusInternalServerError)
		return
	}
	_, err = io.WriteString(w, fmt.Sprintf("%f", workload))
	if err != nil {
		bhLog.LOGGER.Errorf("Fail to write string, reason: %s", err.Error())
	}
}

// returns if the event queue is available or not.
// returns 0 if not available and 1 if available.
func (wh *WebsocketHandle) getEventQueueAvailability() int {
	_, err := wh.EventHandler.GetWorkload()
	if err != nil {
		bhLog.LOGGER.Errorf("eventq is not available, reason %s", err.Error())
		return 0
	}
	return 1
}

// EventReadLoop processes all read requests
func (wh *WebsocketHandle) EventReadLoop(hi hubio.CloudHubIO, info *emodel.HubInfo, stop chan ExitCode) {
	for {
		var msg model.Message
		// set the read timeout as the keepalive interval so that we can disconnect when heart beat is lost
		err := hi.SetReadDeadline(time.Now().Add(time.Duration(wh.EventHandler.KeepaliveInterval) * time.Second))
		if err != nil {
			bhLog.LOGGER.Errorf("SetReadDeadline error, %s", err.Error())
			stop <- hubioReadFail
			return
		}
		_, err = hi.ReadData(&msg)
		if err != nil {
			bhLog.LOGGER.Errorf("read error, connection for node %s will be closed, reason: %s", info.NodeID, err.Error())
			stop <- hubioReadFail
			return
		}
		err = wh.EventHandler.Pub2Controller(info, &msg)
		if err != nil {
			stop <- eventQueueDisconnect
		}
	}
}

// EventWriteLoop processes all write requests
func (wh *WebsocketHandle) EventWriteLoop(hi hubio.CloudHubIO, info *emodel.HubInfo, stop chan ExitCode) {
	wh.EventHandler.eventWriteLoop(hi, info, stop)
}
