package handler

import (
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"k8s.io/klog"

	"github.com/kubeedge/beehive/pkg/core/context"
	"github.com/kubeedge/beehive/pkg/core/model"
	"github.com/kubeedge/kubeedge/cloud/pkg/cloudhub/channelq"
	hubio "github.com/kubeedge/kubeedge/cloud/pkg/cloudhub/common/io"
	emodel "github.com/kubeedge/kubeedge/cloud/pkg/cloudhub/common/model"
	"github.com/kubeedge/kubeedge/cloud/pkg/cloudhub/common/util"
	"github.com/kubeedge/kubeedge/common/constants"
	"github.com/kubeedge/viaduct/pkg/conn"
	"github.com/kubeedge/viaduct/pkg/mux"
)

// ExitCode exit code
type ExitCode int

const (
	hubioReadFail ExitCode = iota
	hubioWriteFail
	eventQueueDisconnect
	nodeStop
)

// constants for error message
const (
	MsgFormatError = "message format not correct"
)

// EventHandle processes events between cloud and edge
type EventHandle struct {
	KeepaliveInterval int
	WriteTimeout      int
	Nodes             sync.Map
	nodeConns         sync.Map
	nodeLocks         sync.Map
	EventQueue        *channelq.ChannelEventQueue
	Context           *context.Context
	Handlers          []HandleFunc
	NodeLimit         int
	KeepaliveChannel  map[string]chan struct{}
}

type HandleFunc func(hi hubio.CloudHubIO, info *emodel.HubInfo, stop chan ExitCode)

var once sync.Once

// CloudhubHandler the shared handler for both websocket and quic servers
var CloudhubHandler *EventHandle

// InitHandler create a handler for websocket and quic servers
func InitHandler(config *util.Config, eventq *channelq.ChannelEventQueue, c *context.Context) {
	once.Do(func() {
		CloudhubHandler = &EventHandle{
			KeepaliveInterval: config.KeepaliveInterval,
			WriteTimeout:      config.WriteTimeout,
			EventQueue:        eventq,
			NodeLimit:         config.NodeLimit,
			Context:           c,
		}

		CloudhubHandler.KeepaliveChannel = make(map[string]chan struct{})
		CloudhubHandler.Handlers = []HandleFunc{CloudhubHandler.KeepaliveCheckLoop, CloudhubHandler.EventWriteLoop}

		CloudhubHandler.initServerEntries()
	})
}

// initServerEntries register handler func
func (eh *EventHandle) initServerEntries() {
	mux.Entry(mux.NewPattern("*").Op("*"), eh.HandleServer)
}

// HandleServer handle all the request from node
func (eh *EventHandle) HandleServer(container *mux.MessageContainer, writer mux.ResponseWriter) {
	nodeID := container.Header.Get("node_id")
	projectID := container.Header.Get("project_id")

	if eh.GetNodeCount() >= eh.NodeLimit {
		klog.Errorf("Fail to serve node %s, reach node limit", nodeID)
		return
	}

	if container.Message.GetOperation() == emodel.OpKeepalive {
		klog.Infof("Keepalive message received from node: %s", nodeID)
		eh.KeepaliveChannel[nodeID] <- struct{}{}
		return
	}

	// handle the reponse from edge
	resourceSplits := strings.Split(container.Message.GetResource(), "/")
	if len(resourceSplits) == 3 {
		if resourceSplits[1] == constants.CSIResourceTypeVolume {
			eh.Context.SendResp(*container.Message)
			return
		}
	}

	err := eh.Pub2Controller(&emodel.HubInfo{ProjectID: projectID, NodeID: nodeID}, container.Message)
	if err != nil {
		// if err, we should stop node, write data to edgehub, stop nodify
		klog.Errorf("Failed to serve handle with error: %s", err.Error())
	}
}

// OnRegister regist node on first connection
func (eh *EventHandle) OnRegister(connection conn.Connection) {
	nodeID := connection.ConnectionState().Headers.Get("node_id")
	projectID := connection.ConnectionState().Headers.Get("project_id")

	if _, ok := eh.KeepaliveChannel[nodeID]; !ok {
		eh.KeepaliveChannel[nodeID] = make(chan struct{}, 1)
	}

	io := &hubio.JSONIO{Connection: connection}
	go eh.ServeConn(io, &emodel.HubInfo{ProjectID: projectID, NodeID: nodeID})
}

// KeepaliveCheckLoop checks whether the edge node is still alive
func (eh *EventHandle) KeepaliveCheckLoop(hi hubio.CloudHubIO, info *emodel.HubInfo, stop chan ExitCode) {
	for {
		keepaliveTimer := time.NewTimer(time.Duration(eh.KeepaliveInterval) * time.Second)
		select {
		case <-eh.KeepaliveChannel[info.NodeID]:
			klog.Infof("Node %s is still alive", info.NodeID)
			keepaliveTimer.Stop()
		case <-keepaliveTimer.C:
			klog.Warningf("Timeout to receive heart beat from edge node %s for project %s",
				info.NodeID, info.ProjectID)
			stop <- nodeStop
			return
		}
	}
}

func dumpEventMetadata(event *emodel.Event) string {
	return fmt.Sprintf("id: %s, parent_id: %s, group: %s, source: %s, resource: %s, operation: %s",
		event.ID, event.ParentID, event.Group, event.Source, event.UserGroup.Resource, event.UserGroup.Operation)
}

func trimMessage(msg *model.Message) {
	resource := msg.GetResource()
	if strings.HasPrefix(resource, emodel.ResNode) {
		tokens := strings.Split(resource, "/")
		if len(tokens) < 3 {
			klog.Warningf("event resource %s starts with node but length less than 3", resource)
		} else {
			msg.SetResourceOperation(strings.Join(tokens[2:], "/"), msg.GetOperation())
		}
	}
}

func notifyEventQueueError(hi hubio.CloudHubIO, code ExitCode, nodeID string) {
	if code == eventQueueDisconnect {
		msg := model.NewMessage("").BuildRouter(emodel.GpResource, emodel.SrcCloudHub, emodel.NewResource(emodel.ResNode, nodeID, nil), emodel.OpDisConnect)
		err := hi.WriteData(msg)
		if err != nil {
			klog.Errorf("fail to notify node %s event queue disconnected, reason: %s", nodeID, err.Error())
		}
	}
}

func constructConnectEvent(info *emodel.HubInfo, isConnected bool) *emodel.Event {
	connected := emodel.OpConnect
	if !isConnected {
		connected = emodel.OpDisConnect
	}
	body := map[string]interface{}{
		"event_type": connected,
		"timestamp":  time.Now().Unix(),
		"client_id":  info.NodeID}
	content, _ := json.Marshal(body)
	msg := model.NewMessage("")
	return &emodel.Event{
		Group:  emodel.GpResource,
		Source: emodel.SrcCloudHub,
		UserGroup: emodel.UserGroupInfo{
			Resource:  emodel.NewResource(emodel.ResNode, info.NodeID, nil),
			Operation: connected,
		},
		ID:        msg.GetID(),
		ParentID:  msg.GetParentID(),
		Timestamp: msg.GetTimestamp(),
		Content:   string(content),
	}
}

func (eh *EventHandle) Pub2Controller(info *emodel.HubInfo, msg *model.Message) error {
	msg.SetResourceOperation(fmt.Sprintf("node/%s/%s", info.NodeID, msg.GetResource()), msg.GetOperation())
	event := emodel.MessageToEvent(msg)
	klog.Infof("event received for node %s %s, content: %s", info.NodeID, dumpEventMetadata(&event), event.Content)
	if event.IsFromEdge() {
		err := eh.EventQueue.Publish(info, &event)
		if err != nil {
			// content is not logged since it may contain sensitive information
			klog.Errorf("fail to publish event for node %s, %s, reason: %s",
				info.NodeID, dumpEventMetadata(&event), err.Error())
			return err
		}
	}
	return nil
}

func (eh *EventHandle) handleNodeQuery(info *emodel.HubInfo, event *emodel.Event) (bool, error) {
	if event.UserGroup.Operation != "request_exist" {
		return false, nil
	}
	msg := model.NewMessage(event.ID)
	event.ID = msg.GetID()
	event.ParentID = msg.GetParentID()
	event.Timestamp = msg.GetTimestamp()
	event.UserGroup.Operation = "response_exist"

	return true, eh.EventQueue.Publish(info, event)
}

func (eh *EventHandle) hubIoWrite(hi hubio.CloudHubIO, nodeID string, v interface{}) error {
	value, ok := eh.nodeLocks.Load(nodeID)
	if !ok {
		return fmt.Errorf("node disconnected")
	}
	mutex := value.(*sync.Mutex)
	mutex.Lock()
	defer mutex.Unlock()

	return hi.WriteData(v)
}

// ServeConn starts serving the incoming connection
func (eh *EventHandle) ServeConn(hi hubio.CloudHubIO, info *emodel.HubInfo) {
	err := eh.EnrollNode(hi, info)
	if err != nil {
		klog.Errorf("fail to enroll node %s, reason %s", info.NodeID, err.Error())
		return
	}

	klog.Infof("edge node %s for project %s connected", info.NodeID, info.ProjectID)
	stop := make(chan ExitCode, 2)

	for _, handle := range eh.Handlers {
		go handle(hi, info, stop)
	}

	code := <-stop
	eh.CancelNode(hi, info, code)
}

// EnrollNode enroll node for the incoming connection
func (eh *EventHandle) EnrollNode(hi hubio.CloudHubIO, info *emodel.HubInfo) error {
	// Wait for the previous connection to be cleaned up
	var err error
	for i := 0; i <= eh.KeepaliveInterval; i++ {
		if err = eh.EventQueue.Connect(info); err == nil {
			break
		}
		time.Sleep(time.Duration(1) * time.Second)
	}
	if err != nil {
		klog.Errorf("fail to connect to event queue for node %s, reason %s", info.NodeID, err.Error())
		notifyEventQueueError(hi, eventQueueDisconnect, info.NodeID)
		err = hi.Close()
		if err != nil {
			klog.Errorf("fail to close connection, reason: %s", err.Error())
		}
		return err
	}

	err = eh.EventQueue.Publish(info, constructConnectEvent(info, true))
	if err != nil {
		klog.Errorf("fail to publish node connect event for node %s, reason %s", info.NodeID, err.Error())
		notifyEventQueueError(hi, eventQueueDisconnect, info.NodeID)
		err = hi.Close()
		if err != nil {
			klog.Errorf("fail to close connection, reason: %s", err.Error())
		}
		eh.EventQueue.Close(info)
		return err
	}

	eh.nodeConns.Store(info.NodeID, hi)
	eh.nodeLocks.Store(info.NodeID, &sync.Mutex{})
	eh.Nodes.Store(info.NodeID, true)
	return nil
}

func (eh *EventHandle) CancelNode(hi hubio.CloudHubIO, info *emodel.HubInfo, code ExitCode) {
	eh.nodeLocks.Delete(info.NodeID)
	eh.nodeConns.Delete(info.NodeID)

	err := eh.EventQueue.Publish(info, constructConnectEvent(info, false))
	if err != nil {
		klog.Errorf("fail to publish node disconnect event for node %s, reason %s", info.NodeID, err.Error())
	}
	notifyEventQueueError(hi, code, info.NodeID)
	eh.Nodes.Delete(info.NodeID)
	err = hi.Close()
	if err != nil {
		klog.Errorf("fail to close connection, reason: %s", err.Error())
	}
	eh.EventQueue.Close(info)
}

// GetNodeCount returns the number of connected Nodes
func (eh *EventHandle) GetNodeCount() int {
	var num int
	iter := func(key, value interface{}) bool {
		num++
		return true
	}
	eh.Nodes.Range(iter)
	return num
}

// GetWorkload returns the workload of the event queue
func (eh *EventHandle) GetWorkload() (float64, error) {
	return eh.EventQueue.Workload()
}

// EventWriteLoop processes all write requests
func (eh *EventHandle) EventWriteLoop(hi hubio.CloudHubIO, info *emodel.HubInfo, stop chan ExitCode) {
	events, err := eh.EventQueue.Consume(info)
	if err != nil {
		klog.Errorf("failed to consume event for node %s, reason: %s", info.NodeID, err.Error())
		stop <- eventQueueDisconnect
		return
	}
	for {
		event, err := events.Get()
		if err != nil {
			klog.Errorf("failed to consume event for node %s, reason: %s", info.NodeID, err.Error())
			if err.Error() == MsgFormatError {
				// error format message should not impact other message
				events.Ack()
				continue
			}
			stop <- eventQueueDisconnect
			return
		}
		isQuery, err := eh.handleNodeQuery(info, event)
		if err != nil {
			klog.Errorf("failed to process node query event for node %s, reason %s", info.NodeID, err.Error())
		}
		if isQuery {
			events.Ack()
			continue
		}

		if event.IsNodeStopped() {
			klog.Infof("node %s is stopped, will disconnect", info.NodeID)
			events.Ack()
			stop <- nodeStop
			return
		}
		if !event.IsToEdge() {
			klog.Infof("skip only to cloud event for node %s, %s, content %s", info.NodeID, dumpEventMetadata(event), event.Content)
			events.Ack()
			continue
		}
		klog.Infof("event to send for node %s, %s, content %s", info.NodeID, dumpEventMetadata(event), event.Content)

		msg := emodel.EventToMessage(event)
		trimMessage(&msg)
		err = hi.SetWriteDeadline(time.Now().Add(time.Duration(eh.WriteTimeout) * time.Second))
		if err != nil {
			klog.Errorf("SetWriteDeadline error, %s", err.Error())
			stop <- hubioWriteFail
			return
		}
		err = eh.hubIoWrite(hi, info.NodeID, &msg)
		if err != nil {
			klog.Errorf("write error, connection for node %s will be closed, affected event %s, reason %s",
				info.NodeID, dumpEventMetadata(event), err.Error())
			stop <- hubioWriteFail
			return
		}
		events.Ack()
	}
}
