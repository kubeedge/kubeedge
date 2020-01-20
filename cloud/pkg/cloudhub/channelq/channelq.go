package channelq

import (
	"fmt"
	"strings"
	"sync"

	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"
	"k8s.io/klog"

	beehiveContext "github.com/kubeedge/beehive/pkg/core/context"
	beehiveModel "github.com/kubeedge/beehive/pkg/core/model"
	"github.com/kubeedge/kubeedge/cloud/pkg/cloudhub/common/model"
	hubconfig "github.com/kubeedge/kubeedge/cloud/pkg/cloudhub/config"
	deviceconstants "github.com/kubeedge/kubeedge/cloud/pkg/devicecontroller/constants"
	edgeconst "github.com/kubeedge/kubeedge/cloud/pkg/edgecontroller/constants"
	edgemessagelayer "github.com/kubeedge/kubeedge/cloud/pkg/edgecontroller/messagelayer"
	"github.com/kubeedge/kubeedge/common/constants"
	common "github.com/kubeedge/kubeedge/common/constants"
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
// node id from it, gets the channel associated with the node
// and pushes the event on the channel
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

		if isListResource(msg.GetResource()) {
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
	nodeListQueue.Add(msg.Header.ID)
	nodeListStore.Add(msg)
}

func (q *ChannelMessageQueue) addMessageToQueue(nodeID string, msg *beehiveModel.Message) {
	if msg.GetResourceVersion() == "" {
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

	// If the message doesn't exist in the store, then compare it with
	// the version stored in the database
	if !exist {
		resourceNamespace, _ := edgemessagelayer.GetNamespace(*msg)
		resourceUID, err := GetMessageUID(msg)
		if err != nil {
			klog.Errorf("fail to get message UID for message: %s", msg.Header.ID)
			return
		}

		objectSync, err := q.ObjectSyncController.ObjectSyncLister.ObjectSyncs(resourceNamespace).Get(strings.Join([]string{nodeID, resourceUID}, "/"))
		if err == nil && msg.GetResourceVersion() <= objectSync.ResourceVersion {
			return
		}
	}

	// Check if message is older than already in store, if it is, discard it directly
	if exist {
		msgInStore := item.(*beehiveModel.Message)
		if msg.GetResourceVersion() <= msgInStore.GetResourceVersion() {
			return
		}
	}

	nodeQueue.Add(messageKey)
	nodeStore.Add(msg)
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

func isListResource(resourceType string) bool {
	if resourceType == beehiveModel.ResourceTypePodlist ||
		resourceType == common.ResourceTypeServiceList ||
		resourceType == common.ResourceTypeEndpointsList ||
		resourceType == beehiveModel.ResourceTypeNode ||
		resourceType == "membership" ||
		strings.Contains(resourceType, deviceconstants.ResourceTypeTwinEdgeUpdated) {
		return true
	}
	return false
}

// getNodeID from "beehive/pkg/core/model".Message.Router.Resource
func GetNodeID(msg *beehiveModel.Message) (string, error) {
	resource := msg.Router.Resource
	tokens := strings.Split(resource, constants.ResourceSep)
	numOfTokens := len(tokens)
	for i, token := range tokens {
		if token == model.ResNode && i+1 < numOfTokens && tokens[i+1] != "" {
			return tokens[i+1], nil
		}
	}

	return "", fmt.Errorf("No nodeID in Message.Router.Resource: %s", resource)
}

// Connect allocates rChannel for given project and group
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
		nodeListStore := cache.NewStore(getMsgKey)
		q.listStorePool.Store(info.NodeID, nodeListStore)
	}
}

// Close closes queues for given node
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

// Publish sends message via the rchannel to Edge Controller
func (q *ChannelMessageQueue) Publish(msg *beehiveModel.Message) error {
	switch msg.Router.Source {
	case model.ResTwin:
		beehiveContext.SendToGroup(model.SrcDeviceController, *msg)
	default:
		beehiveContext.SendToGroup(model.SrcEdgeController, *msg)
	}
	return nil
}

func (q *ChannelMessageQueue) GetNodeQueue(nodeID string) (workqueue.RateLimitingInterface, error) {
	queue, ok := q.queuePool.Load(nodeID)
	if !ok {
		return nil, fmt.Errorf("nodeQueue for edge node %s not found", nodeID)
	}

	nodeQueue := queue.(workqueue.RateLimitingInterface)
	return nodeQueue, nil
}

func (q *ChannelMessageQueue) GetNodeListQueue(nodeID string) (workqueue.RateLimitingInterface, error) {
	queue, ok := q.listQueuePool.Load(nodeID)
	if !ok {
		return nil, fmt.Errorf("nodeListQueue for edge node %s not found", nodeID)
	}

	nodeQueue := queue.(workqueue.RateLimitingInterface)
	return nodeQueue, nil
}

func (q *ChannelMessageQueue) GetNodeStore(nodeID string) (cache.Store, error) {
	store, ok := q.storePool.Load(nodeID)

	if !ok {
		return nil, fmt.Errorf("nodeStore for edge node %s not found", nodeID)
	}

	nodeStore := store.(cache.Store)
	return nodeStore, nil
}

func (q *ChannelMessageQueue) GetNodeListStore(nodeID string) (cache.Store, error) {
	store, ok := q.listStorePool.Load(nodeID)

	if !ok {
		return nil, fmt.Errorf("nodeListStore for edge node %s not found", nodeID)
	}

	nodeStore := store.(cache.Store)
	return nodeStore, nil
}

func GetMessageUID(msg *beehiveModel.Message) (string, error) {
	accessor, err := meta.Accessor(msg.Content)
	if err != nil {
		return "", err
	}

	return string(accessor.GetUID()), nil
}
