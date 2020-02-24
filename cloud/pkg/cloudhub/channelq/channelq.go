package channelq

import (
	"fmt"
	"strings"
	"sync"

	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"
	"k8s.io/klog"

	beehiveContext "github.com/kubeedge/beehive/pkg/core/context"
	beehiveModel "github.com/kubeedge/beehive/pkg/core/model"
	"github.com/kubeedge/kubeedge/cloud/pkg/cloudhub/common/model"
	hubconfig "github.com/kubeedge/kubeedge/cloud/pkg/cloudhub/config"
	edgeconst "github.com/kubeedge/kubeedge/cloud/pkg/edgecontroller/constants"
	edgemessagelayer "github.com/kubeedge/kubeedge/cloud/pkg/edgecontroller/messagelayer"
	"github.com/kubeedge/kubeedge/cloud/pkg/synccontroller"
	commonconst "github.com/kubeedge/kubeedge/common/constants"
)

// ChannelMessageQueue is the channel implementation of MessageQueue
type ChannelMessageQueue struct {
	queuePool sync.Map
	storePool sync.Map

	listQueuePool sync.Map
	listStorePool sync.Map

	ObjectSyncController *hubconfig.ObjectSyncController
}

// NewChannelMessageQueue initializes a new ChannelMessageQueue
func NewChannelMessageQueue(objectSyncController *hubconfig.ObjectSyncController) *ChannelMessageQueue {
	return &ChannelMessageQueue{
		ObjectSyncController: objectSyncController,
	}
}

// DispatchMessage gets the message from the cloud, extracts the
// node id from it, gets the message associated with the node
// and pushes the message to the queue
func (q *ChannelMessageQueue) DispatchMessage() {
	for {
		select {
		case <-beehiveContext.Done():
			klog.Warning("Cloudhub channel eventqueue dispatch message loop stoped")
			return
		default:
		}
		msg, err := beehiveContext.Receive(model.SrcCloudHub)
		if err != nil {
			klog.Info("receive not Message format message")
			continue
		}
		nodeID, err := GetNodeID(&msg)
		if nodeID == "" || err != nil {
			klog.Warning("node id is not found in the message")
			continue
		}

		if isListResource(&msg) {
			q.addListMessageToQueue(nodeID, &msg)
		} else {
			q.addMessageToQueue(nodeID, &msg)
		}
	}
}

func (q *ChannelMessageQueue) addListMessageToQueue(nodeID string, msg *beehiveModel.Message) {
	nodeListQueue, err := q.GetNodeListQueue(nodeID)
	if err != nil {
		klog.Errorf("fail to get nodeListQueue for Node: %s", nodeID)
		return
	}

	nodeListStore, err := q.GetNodeListStore(nodeID)
	if err != nil {
		klog.Errorf("fail to get nodeListStore for Node: %s", nodeID)
		return
	}

	messageKey, _ := getListMsgKey(msg)

	nodeListStore.Add(msg)
	nodeListQueue.Add(messageKey)
}

func (q *ChannelMessageQueue) addMessageToQueue(nodeID string, msg *beehiveModel.Message) {
	if msg.GetResourceVersion() == "" && !isDeleteMessage(msg) {
		return
	}

	nodeQueue, err := q.GetNodeQueue(nodeID)
	if err != nil {
		klog.Errorf("fail to get nodeQueue for Node: %s", nodeID)
		return
	}

	nodeStore, err := q.GetNodeStore(nodeID)
	if err != nil {
		klog.Errorf("fail to get nodeStore for Node: %s", nodeID)
		return
	}

	messageKey, err := getMsgKey(msg)
	if err != nil {
		klog.Errorf("fail to get message key for message: %s", msg.Header.ID)
		return
	}

	item, exist, _ := nodeStore.GetByKey(messageKey)

	if !isDeleteMessage(msg) {
		// If the message doesn't exist in the store, then compare it with
		// the version stored in the database
		if !exist {
			resourceNamespace, _ := edgemessagelayer.GetNamespace(*msg)
			resourceUID, err := GetMessageUID(*msg)
			if err != nil {
				klog.Errorf("fail to get message UID for message: %s", msg.Header.ID)
				return
			}

			objectSync, err := q.ObjectSyncController.ObjectSyncLister.ObjectSyncs(resourceNamespace).Get(synccontroller.BuildObjectSyncName(nodeID, resourceUID))
			if err == nil && msg.GetResourceVersion() <= objectSync.ResourceVersion {
				return
			}
		}

		// Check if message is older than already in store, if it is, discard it directly
		if exist {
			msgInStore := item.(*beehiveModel.Message)
			if msg.GetResourceVersion() <= msgInStore.GetResourceVersion() ||
				isDeleteMessage(msgInStore) {
				return
			}
		}
	}

	nodeStore.Add(msg)
	nodeQueue.Add(messageKey)
}

func getMsgKey(obj interface{}) (string, error) {
	msg := obj.(*beehiveModel.Message)

	if msg.GetGroup() == edgeconst.GroupResource {
		resourceType, _ := edgemessagelayer.GetResourceType(*msg)
		resourceNamespace, _ := edgemessagelayer.GetNamespace(*msg)
		resourceName, _ := edgemessagelayer.GetResourceName(*msg)
		return strings.Join([]string{resourceType, resourceNamespace, resourceName}, "/"), nil
	}

	return "", fmt.Errorf("Failed to get message key")
}

func getListMsgKey(obj interface{}) (string, error) {
	msg := obj.(*beehiveModel.Message)

	return msg.Header.ID, nil
}

func isListResource(msg *beehiveModel.Message) bool {
	msgResource := msg.GetResource()
	if strings.Contains(msgResource, beehiveModel.ResourceTypePodlist) ||
		strings.Contains(msgResource, commonconst.ResourceTypeServiceList) ||
		strings.Contains(msgResource, commonconst.ResourceTypeEndpointsList) ||
		strings.Contains(msgResource, "membership") ||
		strings.Contains(msgResource, "twin/cloud_updated") {
		return true
	}

	if msg.GetOperation() == beehiveModel.ResponseOperation {
		content, ok := msg.Content.(string)
		if ok && content == "OK" {
			return true
		}
	}

	if msg.GetSource() == edgeconst.EdgeControllerModuleName {
		resourceType, _ := edgemessagelayer.GetResourceType(*msg)
		if resourceType == beehiveModel.ResourceTypeNode {
			return true
		}
	}

	return false
}

func isDeleteMessage(msg *beehiveModel.Message) bool {
	deletionTimestamp, err := GetMessageDeletionTimestamp(msg)
	if err != nil {
		klog.Errorf("fail to get message DeletionTimestamp for message: %s", msg.Header.ID)
		return false
	}
	if msg.GetOperation() == beehiveModel.DeleteOperation || deletionTimestamp != nil {
		return true
	}
	return false
}

// GetNodeID from "beehive/pkg/core/model".Message.Router.Resource
func GetNodeID(msg *beehiveModel.Message) (string, error) {
	resource := msg.Router.Resource
	tokens := strings.Split(resource, commonconst.ResourceSep)
	numOfTokens := len(tokens)
	for i, token := range tokens {
		if token == model.ResNode && i+1 < numOfTokens && tokens[i+1] != "" {
			return tokens[i+1], nil
		}
	}

	return "", fmt.Errorf("No nodeID in Message.Router.Resource: %s", resource)
}

// Connect allocates the queues and stores for given node
func (q *ChannelMessageQueue) Connect(info *model.HubInfo) {
	_, queueExist := q.queuePool.Load(info.NodeID)
	_, storeExit := q.storePool.Load(info.NodeID)

	_, listQueueExist := q.listQueuePool.Load(info.NodeID)
	_, listStoreExit := q.listStorePool.Load(info.NodeID)

	if queueExist && storeExit && listQueueExist && listStoreExit {
		klog.Infof("edge node %s is already connected", info.NodeID)
		return
	}

	if !queueExist {
		nodeQueue := workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), info.NodeID)
		q.queuePool.Store(info.NodeID, nodeQueue)
	}
	if !storeExit {
		nodeStore := cache.NewStore(getMsgKey)
		q.storePool.Store(info.NodeID, nodeStore)
	}
	if !listQueueExist {
		nodeListQueue := workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), info.NodeID)
		q.listQueuePool.Store(info.NodeID, nodeListQueue)
	}
	if !listStoreExit {
		nodeListStore := cache.NewStore(getListMsgKey)
		q.listStorePool.Store(info.NodeID, nodeListStore)
	}
}

// Close closes queues and stores for given node
func (q *ChannelMessageQueue) Close(info *model.HubInfo) {
	_, queueExist := q.queuePool.Load(info.NodeID)
	_, storeExist := q.storePool.Load(info.NodeID)

	_, listQueueExist := q.listQueuePool.Load(info.NodeID)
	_, listStoreExit := q.listStorePool.Load(info.NodeID)

	if !queueExist && !storeExist && !listQueueExist && !listStoreExit {
		klog.Warningf("rChannel for edge node %s is already removed", info.NodeID)
		return
	}

	if queueExist {
		q.queuePool.Delete(info.NodeID)
	}
	if storeExist {
		q.storePool.Delete(info.NodeID)
	}
	if listQueueExist {
		q.listQueuePool.Delete(info.NodeID)
	}
	if listStoreExit {
		q.listStorePool.Delete(info.NodeID)
	}
}

// Publish sends message via the rchannel to Controllers
func (q *ChannelMessageQueue) Publish(msg *beehiveModel.Message) error {
	switch msg.Router.Source {
	case model.ResTwin:
		beehiveContext.SendToGroup(model.SrcDeviceController, *msg)
	default:
		beehiveContext.SendToGroup(model.SrcEdgeController, *msg)
	}
	return nil
}

// GetNodeQueue returns the queue for given node
func (q *ChannelMessageQueue) GetNodeQueue(nodeID string) (workqueue.RateLimitingInterface, error) {
	queue, ok := q.queuePool.Load(nodeID)
	if !ok {
		klog.Warningf("nodeQueue for edge node %s not found and created now", nodeID)
		nodeQueue := workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), nodeID)
		q.queuePool.Store(nodeID, nodeQueue)
		return nodeQueue, nil
	}

	nodeQueue := queue.(workqueue.RateLimitingInterface)
	return nodeQueue, nil
}

// GetNodeListQueue returns the listQueue for given node
func (q *ChannelMessageQueue) GetNodeListQueue(nodeID string) (workqueue.RateLimitingInterface, error) {
	queue, ok := q.listQueuePool.Load(nodeID)
	if !ok {
		klog.Warningf("nodeListQueue for edge node %s not found and created now", nodeID)
		nodeListQueue := workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), nodeID)
		q.listQueuePool.Store(nodeID, nodeListQueue)
		return nodeListQueue, nil
	}

	nodeListQueue := queue.(workqueue.RateLimitingInterface)
	return nodeListQueue, nil
}

// GetNodeStore returns the store for given node
func (q *ChannelMessageQueue) GetNodeStore(nodeID string) (cache.Store, error) {
	store, ok := q.storePool.Load(nodeID)
	if !ok {
		klog.Warningf("nodeStore for edge node %s not found and created now", nodeID)
		nodeStore := cache.NewStore(getMsgKey)
		q.storePool.Store(nodeID, nodeStore)
		return nodeStore, nil
	}

	nodeStore := store.(cache.Store)
	return nodeStore, nil
}

// GetNodeListStore returns the listStore for given node
func (q *ChannelMessageQueue) GetNodeListStore(nodeID string) (cache.Store, error) {
	store, ok := q.listStorePool.Load(nodeID)
	if !ok {
		klog.Warningf("nodeListStore for edge node %s not found and created now", nodeID)
		nodeListStore := cache.NewStore(getListMsgKey)
		q.listStorePool.Store(nodeID, nodeListStore)
		return nodeListStore, nil
	}

	nodeListStore := store.(cache.Store)
	return nodeListStore, nil
}

// GetMessageUID returns the UID of the object in message
func GetMessageUID(msg beehiveModel.Message) (string, error) {
	accessor, err := meta.Accessor(msg.Content)
	if err != nil {
		return "", err
	}

	return string(accessor.GetUID()), nil
}

// GetMessageDeletionTimestamp returns the deletionTimestamp of the object in message
func GetMessageDeletionTimestamp(msg *beehiveModel.Message) (*metav1.Time, error) {
	accessor, err := meta.Accessor(msg.Content)
	if err != nil {
		return nil, err
	}

	return accessor.GetDeletionTimestamp(), nil
}
