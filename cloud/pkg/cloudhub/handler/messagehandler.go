package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"sync"
	"time"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/klog/v2"

	beehiveContext "github.com/kubeedge/beehive/pkg/core/context"
	beehiveModel "github.com/kubeedge/beehive/pkg/core/model"
	"github.com/kubeedge/kubeedge/cloud/pkg/apis/reliablesyncs/v1alpha1"
	"github.com/kubeedge/kubeedge/cloud/pkg/cloudhub/channelq"
	hubio "github.com/kubeedge/kubeedge/cloud/pkg/cloudhub/common/io"
	"github.com/kubeedge/kubeedge/cloud/pkg/cloudhub/common/model"
	hubconfig "github.com/kubeedge/kubeedge/cloud/pkg/cloudhub/config"
	deviceconst "github.com/kubeedge/kubeedge/cloud/pkg/devicecontroller/constants"
	edgeconst "github.com/kubeedge/kubeedge/cloud/pkg/edgecontroller/constants"
	edgemessagelayer "github.com/kubeedge/kubeedge/cloud/pkg/edgecontroller/messagelayer"
	"github.com/kubeedge/kubeedge/cloud/pkg/synccontroller"
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
	nodeDisconnect
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
	nodeRegistered    sync.Map
	MessageQueue      *channelq.ChannelMessageQueue
	Handlers          []HandleFunc
	NodeLimit         int
	KeepaliveChannel  sync.Map
	MessageAcks       sync.Map
}

type HandleFunc func(info *model.HubInfo, exitServe chan ExitCode)

var once sync.Once

// CloudhubHandler the shared handler for both websocket and quic servers
var CloudhubHandler *MessageHandle

// InitHandler create a handler for websocket and quic servers
func InitHandler(eventq *channelq.ChannelMessageQueue) {
	once.Do(func() {
		CloudhubHandler = &MessageHandle{
			KeepaliveInterval: int(hubconfig.Config.KeepaliveInterval),
			WriteTimeout:      int(hubconfig.Config.WriteTimeout),
			MessageQueue:      eventq,
			NodeLimit:         int(hubconfig.Config.NodeLimit),
		}

		CloudhubHandler.Handlers = []HandleFunc{
			CloudhubHandler.KeepaliveCheckLoop,
			CloudhubHandler.MessageWriteLoop,
			CloudhubHandler.ListMessageWriteLoop,
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
		klog.V(4).Infof("Keepalive message received from node: %s", nodeID)

		nodeKeepalive, ok := mh.KeepaliveChannel.Load(nodeID)
		if !ok {
			klog.Errorf("Failed to load node : %s", nodeID)
			return
		}
		nodeKeepalive.(chan struct{}) <- struct{}{}
		return
	}

	// handle the response from edge
	if VolumeRegExp.MatchString(container.Message.GetResource()) {
		beehiveContext.SendResp(*container.Message)
		return
	}

	// handle the ack message from edge
	if container.Message.Router.Operation == beehiveModel.ResponseOperation {
		if ackChan, ok := mh.MessageAcks.Load(container.Message.Header.ParentID); ok {
			close(ackChan.(chan struct{}))
			mh.MessageAcks.Delete(container.Message.Header.ParentID)
		}
		return
	}

	err := mh.PubToController(&model.HubInfo{ProjectID: projectID, NodeID: nodeID}, container.Message)
	if err != nil {
		// if err, we should stop node, write data to edgehub, stop nodify
		klog.Errorf("Failed to serve handle with error: %s", err.Error())
	}
}

// OnRegister register node on first connection
func (mh *MessageHandle) OnRegister(connection conn.Connection) {
	nodeID := connection.ConnectionState().Headers.Get("node_id")
	projectID := connection.ConnectionState().Headers.Get("project_id")

	if _, ok := mh.KeepaliveChannel.Load(nodeID); !ok {
		mh.KeepaliveChannel.Store(nodeID, make(chan struct{}, 1))
	}

	io := &hubio.JSONIO{Connection: connection}

	if _, ok := mh.nodeRegistered.Load(nodeID); ok {
		if conn, exist := mh.nodeConns.Load(nodeID); exist {
			conn.(hubio.CloudHubIO).Close()
		}
		mh.nodeConns.Store(nodeID, io)
		return
	}
	mh.nodeConns.Store(nodeID, io)
	go mh.ServeConn(&model.HubInfo{ProjectID: projectID, NodeID: nodeID})
}

// KeepaliveCheckLoop checks whether the edge node is still alive
func (mh *MessageHandle) KeepaliveCheckLoop(info *model.HubInfo, stopServe chan ExitCode) {
	keepaliveTicker := time.NewTimer(time.Duration(mh.KeepaliveInterval) * time.Second)
	nodeKeepaliveChannel, _ := mh.KeepaliveChannel.Load(info.NodeID)

	for {
		select {
		case _, ok := <-nodeKeepaliveChannel.(chan struct{}):
			if !ok {
				klog.Warningf("Stop keepalive check for node: %s", info.NodeID)
				return
			}

			// Reset is called after Stop or expired timer
			if !keepaliveTicker.Stop() {
				select {
				case <-keepaliveTicker.C:
				default:
				}
			}
			klog.V(4).Infof("Node %s is still alive", info.NodeID)
			keepaliveTicker.Reset(time.Duration(mh.KeepaliveInterval) * time.Second)
		case <-keepaliveTicker.C:
			if conn, ok := mh.nodeConns.Load(info.NodeID); ok {
				klog.Warningf("Timeout to receive heart beat from edge node %s for project %s", info.NodeID, info.ProjectID)
				conn.(hubio.CloudHubIO).Close()
				mh.nodeConns.Delete(info.NodeID)
			}
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
	klog.V(4).Infof("event received for node %s %s, content: %s", info.NodeID, dumpMessageMetadata(msg), msg.Content)
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
func (mh *MessageHandle) ServeConn(info *model.HubInfo) {
	err := mh.RegisterNode(info)
	if err != nil {
		klog.Errorf("fail to register node %s, reason %s", info.NodeID, err.Error())
		return
	}

	klog.Infof("edge node %s for project %s connected", info.NodeID, info.ProjectID)
	exitServe := make(chan ExitCode, 3)

	for _, handle := range mh.Handlers {
		go handle(info, exitServe)
	}

	code := <-exitServe
	mh.UnregisterNode(info, code)
}

// RegisterNode register node in cloudhub for the incoming connection
func (mh *MessageHandle) RegisterNode(info *model.HubInfo) error {
	hi, err := mh.getNodeConnection(info.NodeID)
	if err != nil {
		return err
	}
	mh.MessageQueue.Connect(info)

	err = mh.MessageQueue.Publish(constructConnectMessage(info, true))
	if err != nil {
		klog.Errorf("fail to publish node connect event for node %s, reason %s", info.NodeID, err.Error())
		notifyEventQueueError(hi, messageQueueDisconnect, info.NodeID)
		err = hi.Close()
		if err != nil {
			klog.Errorf("fail to close connection, reason: %s", err.Error())
		}
		return err
	}

	mh.nodeLocks.Store(info.NodeID, &sync.Mutex{})
	mh.Nodes.Store(info.NodeID, true)
	mh.nodeRegistered.Store(info.NodeID, true)
	return nil
}

// UnregisterNode unregister node in cloudhub
func (mh *MessageHandle) UnregisterNode(info *model.HubInfo, code ExitCode) {
	if hi, err := mh.getNodeConnection(info.NodeID); err == nil {
		notifyEventQueueError(hi, code, info.NodeID)
		hi.Close()
	}

	mh.nodeLocks.Delete(info.NodeID)
	mh.nodeConns.Delete(info.NodeID)
	mh.nodeRegistered.Delete(info.NodeID)
	nodeKeepalive, ok := mh.KeepaliveChannel.Load(info.NodeID)
	if !ok {
		klog.Errorf("fail to load node %s", info.NodeID)
	} else {
		close(nodeKeepalive.(chan struct{}))
		mh.KeepaliveChannel.Delete(info.NodeID)
	}

	err := mh.MessageQueue.Publish(constructConnectMessage(info, false))
	if err != nil {
		klog.Errorf("fail to publish node disconnect event for node %s, reason %s", info.NodeID, err.Error())
	}

	mh.Nodes.Delete(info.NodeID)

	if err != nil {
		klog.Errorf("fail to close connection, reason: %s", err.Error())
	}

	// delete the nodeQueue and nodeStore when node stopped
	if code == nodeStop {
		mh.MessageQueue.Close(info)
	}
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

// ListMessageWriteLoop processes all list type resource write requests
func (mh *MessageHandle) ListMessageWriteLoop(info *model.HubInfo, stopServe chan ExitCode) {
	nodeListQueue := mh.MessageQueue.GetNodeListQueue(info.NodeID)
	nodeListStore := mh.MessageQueue.GetNodeListStore(info.NodeID)
	nodeQueue := mh.MessageQueue.GetNodeQueue(info.NodeID)

	for {
		key, quit := nodeListQueue.Get()
		if quit {
			klog.Errorf("nodeListQueue for node %s has shutdown", info.NodeID)
			return
		}

		obj, exist, _ := nodeListStore.GetByKey(key.(string))
		if !exist {
			klog.Errorf("nodeListStore for node %s doesn't exist", info.NodeID)
			continue
		}
		msg := obj.(*beehiveModel.Message)

		if model.IsNodeStopped(msg) {
			klog.Warningf("node %s is deleted, data for node will be cleaned up", info.NodeID)
			nodeQueue.ShutDown()
			nodeListQueue.ShutDown()
			stopServe <- nodeStop
			return
		}
		if !model.IsToEdge(msg) {
			klog.Infof("skip only to cloud event for node %s, %s, content %s", info.NodeID, dumpMessageMetadata(msg), msg.Content)
			continue
		}
		klog.V(4).Infof("event to send for node %s, %s, content %s", info.NodeID, dumpMessageMetadata(msg), msg.Content)

		trimMessage(msg)

		conn, ok := mh.nodeConns.Load(info.NodeID)
		if !ok {
			continue
		}

		if err := mh.send(conn.(hubio.CloudHubIO), info, msg); err != nil {
			klog.Errorf("failed to send to cloudhub, err: %v", err)
		}

		// delete successfully sent events from the queue/store
		if err := nodeListStore.Delete(msg); err != nil {
			klog.Errorf("failed to delete msg from store, err: %v", err)
		}

		nodeListQueue.Forget(key.(string))
		nodeListQueue.Done(key)
	}
}

// MessageWriteLoop processes all write requests
func (mh *MessageHandle) MessageWriteLoop(info *model.HubInfo, stopServe chan ExitCode) {
	nodeQueue := mh.MessageQueue.GetNodeQueue(info.NodeID)
	nodeStore := mh.MessageQueue.GetNodeStore(info.NodeID)

	for {
		key, quit := nodeQueue.Get()
		if quit {
			klog.Errorf("nodeQueue for node %s has shutdown", info.NodeID)
			return
		}

		obj, exist, _ := nodeStore.GetByKey(key.(string))
		if !exist {
			klog.Errorf("nodeStore for node %s doesn't exist", info.NodeID)
			nodeQueue.Done(key)
			continue
		}
		msg := obj.(*beehiveModel.Message)

		if !model.IsToEdge(msg) {
			klog.Infof("skip only to cloud event for node %s, %s, content %s", info.NodeID, dumpMessageMetadata(msg), msg.Content)
			nodeQueue.Done(key)
			continue
		}
		klog.V(4).Infof("event to send for node %s, %s, content %s", info.NodeID, dumpMessageMetadata(msg), msg.Content)

		copyMsg := deepcopy(msg)
		trimMessage(copyMsg)

		for {
			conn, ok := mh.nodeConns.Load(info.NodeID)
			if !ok {
				time.Sleep(time.Second * 2)
				continue
			}
			err := mh.sendMsg(conn.(hubio.CloudHubIO), info, copyMsg, msg, nodeStore)
			if err != nil {
				klog.Errorf("Failed to send event to node: %s, affected event: %s, err: %s",
					info.NodeID, dumpMessageMetadata(copyMsg), err.Error())
				nodeQueue.AddRateLimited(key.(string))
				time.Sleep(time.Second * 2)
			}
			break
		}

		nodeQueue.Forget(key.(string))
		nodeQueue.Done(key)
	}
}

func (mh *MessageHandle) sendMsg(hi hubio.CloudHubIO, info *model.HubInfo, copyMsg, msg *beehiveModel.Message, nodeStore cache.Store) error {
	ackChan := make(chan struct{})
	mh.MessageAcks.Store(copyMsg.GetID(), ackChan)

	// initialize timer and retry count for sending message
	var (
		retry                       = 0
		retryInterval time.Duration = 5
	)
	ticker := time.NewTimer(retryInterval * time.Second)
	err := mh.send(hi, info, copyMsg)
	if err != nil {
		return err
	}

LOOP:
	for {
		select {
		case <-ackChan:
			mh.saveSuccessPoint(msg, info, nodeStore)
			break LOOP
		case <-ticker.C:
			if retry == 4 {
				break LOOP
			}
			err := mh.send(hi, info, copyMsg)
			if err != nil {
				return err
			}
			retry++
			ticker.Reset(time.Second * retryInterval)
		}
	}
	return nil
}

func (mh *MessageHandle) send(hi hubio.CloudHubIO, info *model.HubInfo, msg *beehiveModel.Message) error {
	err := mh.hubIoWrite(hi, info.NodeID, msg)
	if err != nil {
		return err
	}
	return nil
}

func (mh *MessageHandle) saveSuccessPoint(msg *beehiveModel.Message, info *model.HubInfo, nodeStore cache.Store) {
	if msg.GetGroup() == edgeconst.GroupResource {
		resourceNamespace, _ := edgemessagelayer.GetNamespace(*msg)
		resourceName, _ := edgemessagelayer.GetResourceName(*msg)
		resourceType, _ := edgemessagelayer.GetResourceType(*msg)
		resourceUID, err := channelq.GetMessageUID(*msg)
		if err != nil {
			klog.Errorf("failed to get message UID %v, err: %v", msg, err)
			return
		}

		objectSyncName := synccontroller.BuildObjectSyncName(info.NodeID, resourceUID)

		if msg.GetOperation() == beehiveModel.DeleteOperation {
			if err := nodeStore.Delete(msg); err != nil {
				klog.Errorf("failed to delete message %v, err: %v", msg, err)
				return
			}
			mh.deleteSuccessPoint(resourceNamespace, objectSyncName)
			return
		}

		objectSync, err := mh.MessageQueue.ObjectSyncController.CrdClient.ReliablesyncsV1alpha1().ObjectSyncs(resourceNamespace).Get(context.Background(), objectSyncName, metav1.GetOptions{})
		if err == nil {
			objectSync.Status.ObjectResourceVersion = msg.GetResourceVersion()
			_, err := mh.MessageQueue.ObjectSyncController.CrdClient.ReliablesyncsV1alpha1().ObjectSyncs(resourceNamespace).UpdateStatus(context.Background(), objectSync, metav1.UpdateOptions{})
			if err != nil {
				klog.Errorf("Failed to update objectSync: %v, resourceType: %s, resourceNamespace: %s, resourceName: %s",
					err, resourceType, resourceNamespace, resourceName)
			}
		} else if err != nil && apierrors.IsNotFound(err) {
			objectSync := &v1alpha1.ObjectSync{
				ObjectMeta: metav1.ObjectMeta{
					Name: objectSyncName,
				},
				Spec: v1alpha1.ObjectSyncSpec{
					ObjectAPIVersion: "",
					ObjectKind:       resourceType,
					ObjectName:       resourceName,
				},
			}
			_, err := mh.MessageQueue.ObjectSyncController.CrdClient.ReliablesyncsV1alpha1().ObjectSyncs(resourceNamespace).Create(context.Background(), objectSync, metav1.CreateOptions{})
			if err != nil {
				klog.Errorf("Failed to create objectSync: %s, err: %v", objectSyncName, err)
				return
			}

			objectSyncStatus, err := mh.MessageQueue.ObjectSyncController.CrdClient.ReliablesyncsV1alpha1().ObjectSyncs(resourceNamespace).Get(context.Background(), objectSyncName, metav1.GetOptions{})
			if err != nil {
				klog.Errorf("Failed to get objectSync: %s, err: %v", objectSyncName, err)
			}
			objectSyncStatus.Status.ObjectResourceVersion = msg.GetResourceVersion()
			if _, err := mh.MessageQueue.ObjectSyncController.CrdClient.ReliablesyncsV1alpha1().ObjectSyncs(resourceNamespace).UpdateStatus(context.Background(), objectSyncStatus, metav1.UpdateOptions{}); err != nil {
				klog.Errorf("Failed to update objectSync: %s, err: %v", objectSyncName, err)
			}
		}
	}

	// TODO: save device info
	if msg.GetGroup() == deviceconst.GroupTwin {
	}
	klog.V(4).Infof("saveSuccessPoint successfully for message: %s", msg.GetResource())
}

func (mh *MessageHandle) deleteSuccessPoint(resourceNamespace, objectSyncName string) {
	if err := mh.MessageQueue.ObjectSyncController.CrdClient.ReliablesyncsV1alpha1().ObjectSyncs(resourceNamespace).Delete(context.Background(), objectSyncName, *metav1.NewDeleteOptions(0)); err != nil {
		klog.Errorf("Delete Success Point failed with error: %v", err)
	}
}

func (mh *MessageHandle) getNodeConnection(nodeid string) (hubio.CloudHubIO, error) {
	conn, ok := mh.nodeConns.Load(nodeid)
	if !ok {
		return nil, fmt.Errorf("Failed to get connection for node: %s", nodeid)
	}

	return conn.(hubio.CloudHubIO), nil
}

func deepcopy(msg *beehiveModel.Message) *beehiveModel.Message {
	if msg == nil {
		return nil
	}
	out := new(beehiveModel.Message)
	out.Header = msg.Header
	out.Router = msg.Router
	out.Content = msg.Content
	return out
}
