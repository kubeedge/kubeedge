package handler

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"sync"
	"time"

	"k8s.io/klog"

	beehiveContext "github.com/kubeedge/beehive/pkg/core/context"
	beehiveModel "github.com/kubeedge/beehive/pkg/core/model"
	"github.com/kubeedge/kubeedge/cloud/pkg/cloudhub/channelq"
	hubio "github.com/kubeedge/kubeedge/cloud/pkg/cloudhub/common/io"
	"github.com/kubeedge/kubeedge/cloud/pkg/cloudhub/common/model"
	hubconfig "github.com/kubeedge/kubeedge/cloud/pkg/cloudhub/config"
	"github.com/kubeedge/kubeedge/common/constants"
	"github.com/kubeedge/viaduct/pkg/conn"
	"github.com/kubeedge/viaduct/pkg/mux"
)

// ExitCode exit code
type ExitCode int

const (
	hubioReadFail ExitCode = iota
	hubioWriteFail
	messageQueueDisconnect
	nodeStop
)

// constants for error message
const (
	MsgFormatError = "message format not correct"
	VolumePattern  = `^\w[-\w.+]*/` + constants.CSIResourceTypeVolume + `/\w[-\w.+]*`
)

// VolumeRegExp is used to validate the volume resource
var VolumeRegExp = regexp.MustCompile(VolumePattern)

// MessageHandle processes messages between cloud and edge
type MessageHandle struct {
	KeepaliveInterval int
	WriteTimeout      int
	Nodes             sync.Map
	nodeConns         sync.Map
	nodeLocks         sync.Map
	MessageQueue      *channelq.ChannelMessageQueue
	Handlers          []HandleFunc
	NodeLimit         int
	KeepaliveChannel  map[string]chan struct{}
}

type HandleFunc func(hi hubio.CloudHubIO, info *model.HubInfo, stop chan ExitCode)

var once sync.Once

// CloudhubHandler the shared handler for both websocket and quic servers
var CloudhubHandler *MessageHandle

// InitHandler create a handler for websocket and quic servers
func InitHandler(config *hubconfig.Configure, eventq *channelq.ChannelMessageQueue) {
	once.Do(func() {
		CloudhubHandler = &MessageHandle{
			KeepaliveInterval: config.KeepaliveInterval,
			WriteTimeout:      config.WriteTimeout,
			MessageQueue:      eventq,
			NodeLimit:         config.NodeLimit,
		}

		CloudhubHandler.KeepaliveChannel = make(map[string]chan struct{})
		CloudhubHandler.Handlers = []HandleFunc{
			CloudhubHandler.KeepaliveCheckLoop,
			CloudhubHandler.MessageWriteLoop,
		}

		CloudhubHandler.initServerEntries()
	})
}

// initServerEntries register handler func
func (mh *MessageHandle) initServerEntries() {
	mux.Entry(mux.NewPattern("*").Op("*"), mh.HandleServer)
}

// HandleServer handle all the request from node
func (mh *MessageHandle) HandleServer(container *mux.MessageContainer, writer mux.ResponseWriter) {
	nodeID := container.Header.Get("node_id")
	projectID := container.Header.Get("project_id")

	if mh.GetNodeCount() >= mh.NodeLimit {
		klog.Errorf("Fail to serve node %s, reach node limit", nodeID)
		return
	}

	if container.Message.GetOperation() == model.OpKeepalive {
		klog.Infof("Keepalive message received from node: %s", nodeID)
		mh.KeepaliveChannel[nodeID] <- struct{}{}
		return
	}

	// handle the reponse from edge
	if VolumeRegExp.MatchString(container.Message.GetResource()) {
		beehiveContext.SendResp(*container.Message)
		return
	}

	err := mh.PubToController(&model.HubInfo{ProjectID: projectID, NodeID: nodeID}, container.Message)
	if err != nil {
		// if err, we should stop node, write data to edgehub, stop nodify
		klog.Errorf("Failed to serve handle with error: %s", err.Error())
	}
}

// OnRegister regist node on first connection
func (mh *MessageHandle) OnRegister(connection conn.Connection) {
	nodeID := connection.ConnectionState().Headers.Get("node_id")
	projectID := connection.ConnectionState().Headers.Get("project_id")

	if _, ok := mh.KeepaliveChannel[nodeID]; !ok {
		mh.KeepaliveChannel[nodeID] = make(chan struct{}, 1)
	}

	io := &hubio.JSONIO{Connection: connection}
	go mh.ServeConn(io, &model.HubInfo{ProjectID: projectID, NodeID: nodeID})
}

// KeepaliveCheckLoop checks whether the edge node is still alive
func (mh *MessageHandle) KeepaliveCheckLoop(hi hubio.CloudHubIO, info *model.HubInfo, stop chan ExitCode) {
	for {
		keepaliveTimer := time.NewTimer(time.Duration(mh.KeepaliveInterval) * time.Second)
		select {
		case <-mh.KeepaliveChannel[info.NodeID]:
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

func dumpMessageMetadata(msg *beehiveModel.Message) string {
	return fmt.Sprintf("id: %s, parent_id: %s, group: %s, source: %s, resource: %s, operation: %s",
		msg.Header.ID, msg.Header.ParentID, msg.Router.Group, msg.Router.Source, msg.Router.Resource, msg.Router.Operation)
}

func trimMessage(msg *beehiveModel.Message) {
	resource := msg.GetResource()
	if strings.HasPrefix(resource, model.ResNode) {
		tokens := strings.Split(resource, "/")
		if len(tokens) < 3 {
			klog.Warningf("event resource %s starts with node but length less than 3", resource)
		} else {
			msg.SetResourceOperation(strings.Join(tokens[2:], "/"), msg.GetOperation())
		}
	}
}

func notifyEventQueueError(hi hubio.CloudHubIO, code ExitCode, nodeID string) {
	if code == messageQueueDisconnect {
		msg := beehiveModel.NewMessage("").BuildRouter(model.GpResource, model.SrcCloudHub, model.NewResource(model.ResNode, nodeID, nil), model.OpDisConnect)
		err := hi.WriteData(msg)
		if err != nil {
			klog.Errorf("fail to notify node %s event queue disconnected, reason: %s", nodeID, err.Error())
		}
	}
}

func constructConnectMessage(info *model.HubInfo, isConnected bool) *beehiveModel.Message {
	connected := model.OpConnect
	if !isConnected {
		connected = model.OpDisConnect
	}
	body := map[string]interface{}{
		"event_type": connected,
		"timestamp":  time.Now().Unix(),
		"client_id":  info.NodeID}
	content, _ := json.Marshal(body)

	msg := beehiveModel.NewMessage("")
	msg.BuildRouter(model.SrcCloudHub, model.GpResource, model.NewResource(model.ResNode, info.NodeID, nil), connected)
	msg.FillBody(content)
	return msg
}

func (mh *MessageHandle) PubToController(info *model.HubInfo, msg *beehiveModel.Message) error {
	msg.SetResourceOperation(fmt.Sprintf("node/%s/%s", info.NodeID, msg.GetResource()), msg.GetOperation())
	klog.Infof("event received for node %s %s, content: %s", info.NodeID, dumpMessageMetadata(msg), msg.Content)
	if model.IsFromEdge(msg) {
		err := mh.MessageQueue.Publish(msg)
		if err != nil {
			// content is not logged since it may contain sensitive information
			klog.Errorf("fail to publish event for node %s, %s, reason: %s",
				info.NodeID, dumpMessageMetadata(msg), err.Error())
			return err
		}
	}
	return nil
}

func (mh *MessageHandle) hubIoWrite(hi hubio.CloudHubIO, nodeID string, msg *beehiveModel.Message) error {
	value, ok := mh.nodeLocks.Load(nodeID)
	if !ok {
		return fmt.Errorf("node disconnected")
	}
	mutex := value.(*sync.Mutex)
	mutex.Lock()
	defer mutex.Unlock()

	return hi.WriteData(msg)
}

// ServeConn starts serving the incoming connection
func (mh *MessageHandle) ServeConn(hi hubio.CloudHubIO, info *model.HubInfo) {
	err := mh.EnrollNode(hi, info)
	if err != nil {
		klog.Errorf("fail to enroll node %s, reason %s", info.NodeID, err.Error())
		return
	}

	klog.Infof("edge node %s for project %s connected", info.NodeID, info.ProjectID)
	stop := make(chan ExitCode, 2)

	for _, handle := range mh.Handlers {
		go handle(hi, info, stop)
	}

	code := <-stop
	mh.CancelNode(hi, info, code)
}

// EnrollNode enroll node for the incoming connection
func (mh *MessageHandle) EnrollNode(hi hubio.CloudHubIO, info *model.HubInfo) error {
	// Wait for the previous connection to be cleaned up
	var err error
	for i := 0; i <= mh.KeepaliveInterval; i++ {
		if err = mh.MessageQueue.Connect(info); err == nil {
			break
		}
		time.Sleep(time.Duration(1) * time.Second)
	}
	if err != nil {
		klog.Errorf("fail to connect to event queue for node %s, reason %s", info.NodeID, err.Error())
		notifyEventQueueError(hi, messageQueueDisconnect, info.NodeID)
		err = hi.Close()
		if err != nil {
			klog.Errorf("fail to close connection, reason: %s", err.Error())
		}
		return err
	}

	err = mh.MessageQueue.Publish(constructConnectMessage(info, true))
	if err != nil {
		klog.Errorf("fail to publish node connect event for node %s, reason %s", info.NodeID, err.Error())
		notifyEventQueueError(hi, messageQueueDisconnect, info.NodeID)
		err = hi.Close()
		if err != nil {
			klog.Errorf("fail to close connection, reason: %s", err.Error())
		}
		mh.MessageQueue.Close(info)
		return err
	}

	mh.nodeConns.Store(info.NodeID, hi)
	mh.nodeLocks.Store(info.NodeID, &sync.Mutex{})
	mh.Nodes.Store(info.NodeID, true)
	return nil
}

func (mh *MessageHandle) CancelNode(hi hubio.CloudHubIO, info *model.HubInfo, code ExitCode) {
	mh.nodeLocks.Delete(info.NodeID)
	mh.nodeConns.Delete(info.NodeID)

	err := mh.MessageQueue.Publish(constructConnectMessage(info, false))
	if err != nil {
		klog.Errorf("fail to publish node disconnect event for node %s, reason %s", info.NodeID, err.Error())
	}
	notifyEventQueueError(hi, code, info.NodeID)
	mh.Nodes.Delete(info.NodeID)
	err = hi.Close()
	if err != nil {
		klog.Errorf("fail to close connection, reason: %s", err.Error())
	}
	mh.MessageQueue.Close(info)
}

// GetNodeCount returns the number of connected Nodes
func (mh *MessageHandle) GetNodeCount() int {
	var num int
	iter := func(key, value interface{}) bool {
		num++
		return true
	}
	mh.Nodes.Range(iter)
	return num
}

// GetWorkload returns the workload of the event queue
func (mh *MessageHandle) GetWorkload() (float64, error) {
	return mh.MessageQueue.Workload()
}

// MessageWriteLoop processes all write requests
func (mh *MessageHandle) MessageWriteLoop(hi hubio.CloudHubIO, info *model.HubInfo, stop chan ExitCode) {
	messages, err := mh.MessageQueue.Consume(info)
	if err != nil {
		klog.Errorf("failed to consume event for node %s, reason: %s", info.NodeID, err.Error())
		stop <- messageQueueDisconnect
		return
	}
	for {
		msg, err := messages.Get()
		if err != nil {
			klog.Errorf("failed to consume event for node %s, reason: %s", info.NodeID, err.Error())
			if err.Error() == MsgFormatError {
				// error format message should not impact other message
				messages.Ack()
				continue
			}
			stop <- messageQueueDisconnect
			return
		}

		if model.IsNodeStopped(msg) {
			klog.Infof("node %s is stopped, will disconnect", info.NodeID)
			messages.Ack()
			stop <- nodeStop
			return
		}
		if !model.IsToEdge(msg) {
			klog.Infof("skip only to cloud event for node %s, %s, content %s", info.NodeID, dumpMessageMetadata(msg), msg.Content)
			messages.Ack()
			continue
		}
		klog.Infof("event to send for node %s, %s, content %s", info.NodeID, dumpMessageMetadata(msg), msg.Content)

		trimMessage(msg)
		err = hi.SetWriteDeadline(time.Now().Add(time.Duration(mh.WriteTimeout) * time.Second))
		if err != nil {
			klog.Errorf("SetWriteDeadline error, %s", err.Error())
			stop <- hubioWriteFail
			return
		}
		err = mh.hubIoWrite(hi, info.NodeID, msg)
		if err != nil {
			klog.Errorf("write error, connection for node %s will be closed, affected event %s, reason %s",
				info.NodeID, dumpMessageMetadata(msg), err.Error())
			stop <- hubioWriteFail
			return
		}
		messages.Ack()
	}
}
