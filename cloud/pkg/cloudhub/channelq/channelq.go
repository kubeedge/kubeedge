package channelq

import (
	"context"
	"fmt"
	"strings"
	"sync"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"
	"k8s.io/klog/v2"

	beehiveContext "github.com/kubeedge/beehive/pkg/core/context"
	beehiveModel "github.com/kubeedge/beehive/pkg/core/model"
	"github.com/kubeedge/kubeedge/cloud/pkg/cloudhub/common"
	"github.com/kubeedge/kubeedge/cloud/pkg/cloudhub/common/model"
	"github.com/kubeedge/kubeedge/cloud/pkg/common/messagelayer"
	"github.com/kubeedge/kubeedge/cloud/pkg/common/modules"
	"github.com/kubeedge/kubeedge/cloud/pkg/dynamiccontroller/application"
	edgeconst "github.com/kubeedge/kubeedge/cloud/pkg/edgecontroller/constants"
	"github.com/kubeedge/kubeedge/cloud/pkg/synccontroller"
	commonconst "github.com/kubeedge/kubeedge/common/constants"
	"github.com/kubeedge/kubeedge/pkg/apis/reliablesyncs/v1alpha1"
	crdClientset "github.com/kubeedge/kubeedge/pkg/client/clientset/versioned"
	reliablesyncslisters "github.com/kubeedge/kubeedge/pkg/client/listers/reliablesyncs/v1alpha1"
	"github.com/kubeedge/kubeedge/pkg/metaserver/util"
)

// ChannelMessageQueue is the channel implementation of MessageQueue
type ChannelMessageQueue struct {
	queuePool sync.Map
	storePool sync.Map

	listQueuePool sync.Map
	listStorePool sync.Map

	objectSyncLister        reliablesyncslisters.ObjectSyncLister
	clusterObjectSyncLister reliablesyncslisters.ClusterObjectSyncLister

	crdClient crdClientset.Interface
}

// NewChannelMessageQueue initializes a new ChannelMessageQueue
func NewChannelMessageQueue(objectSyncLister reliablesyncslisters.ObjectSyncLister, clusterObjectSyncLister reliablesyncslisters.ClusterObjectSyncLister, crdClient crdClientset.Interface) *ChannelMessageQueue {
	return &ChannelMessageQueue{
		objectSyncLister:        objectSyncLister,
		clusterObjectSyncLister: clusterObjectSyncLister,
		crdClient:               crdClient,
	}
}

// DispatchMessage gets the message from the cloud, extracts the
// node id from it, gets the message associated with the node
// and pushes the message to the queue
func (q *ChannelMessageQueue) DispatchMessage() {
	for {
		select {
		case <-beehiveContext.Done():
			klog.Warning("Cloudhub channel eventqueue dispatch message loop stopped")
			return
		default:
		}
		msg, err := beehiveContext.Receive(model.SrcCloudHub)
		klog.V(4).Infof("[cloudhub] dispatchMessage to edge: %+v", msg)
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
	nodeListQueue := q.GetNodeListQueue(nodeID)
	nodeListStore := q.GetNodeListStore(nodeID)

	messageKey, _ := getListMsgKey(msg)

	if err := nodeListStore.Add(msg); err != nil {
		klog.Errorf("failed to add msg: %s", err)
		return
	}
	nodeListQueue.Add(messageKey)
}

func (q *ChannelMessageQueue) addMessageToQueue(nodeID string, msg *beehiveModel.Message) {
	if msg.GetResourceVersion() == "" && !isDeleteMessage(msg) {
		return
	}

	nodeQueue := q.GetNodeQueue(nodeID)
	nodeStore := q.GetNodeStore(nodeID)

	messageKey, err := getMsgKey(msg)
	if err != nil {
		klog.Errorf("fail to get message key for message: %s", msg.Header.ID)
		return
	}

	//if the operation is delete, force to sync the resource message
	//if the operation is response, force to sync the resource message, since the edgecore requests it
	if !isDeleteMessage(msg) && msg.GetOperation() != beehiveModel.ResponseOperation {
		item, exist, _ := nodeStore.GetByKey(messageKey)
		// If the message doesn't exist in the store, then compare it with
		// the version stored in the database
		if !exist {
			resourceNamespace, _ := messagelayer.GetNamespace(*msg)
			resourceName, _ := messagelayer.GetResourceName(*msg)
			resourceUID, err := common.GetMessageUID(*msg)
			objectSyncName := synccontroller.BuildObjectSyncName(nodeID, resourceUID)
			if err != nil {
				klog.Errorf("fail to get message UID for message: %s", msg.Header.ID)
				return
			}
			objectSync, err := q.objectSyncLister.ObjectSyncs(resourceNamespace).Get(objectSyncName)
			if err == nil && objectSync.Status.ObjectResourceVersion != "" && synccontroller.CompareResourceVersion(msg.GetResourceVersion(), objectSync.Status.ObjectResourceVersion) <= 0 {
				return
			} else if err != nil && apierrors.IsNotFound(err) {
				objectSync := &v1alpha1.ObjectSync{
					ObjectMeta: metav1.ObjectMeta{
						Name: objectSyncName,
					},
					Spec: v1alpha1.ObjectSyncSpec{
						ObjectAPIVersion: util.GetMessageAPIVersion(msg),
						ObjectKind:       util.GetMessageResourceType(msg),
						ObjectName:       resourceName,
					},
				}
				objectSyncStatus, err := q.crdClient.ReliablesyncsV1alpha1().ObjectSyncs(resourceNamespace).Create(context.Background(), objectSync, metav1.CreateOptions{})
				if err != nil {
					klog.Errorf("Failed to create objectSync: %s, err: %v", objectSyncName, err)
					return
				}
				objectSyncStatus.Status.ObjectResourceVersion = "0"
				if _, err := q.crdClient.ReliablesyncsV1alpha1().ObjectSyncs(resourceNamespace).UpdateStatus(context.Background(), objectSyncStatus, metav1.UpdateOptions{}); err != nil {
					klog.Errorf("Failed to update objectSync: %s, err: %v", objectSyncName, err)
				}
			}
		} else {
			// Check if message is older than already in store, if it is, discard it directly
			msgInStore := item.(*beehiveModel.Message)
			if isDeleteMessage(msgInStore) || synccontroller.CompareResourceVersion(msg.GetResourceVersion(), msgInStore.GetResourceVersion()) <= 0 {
				return
			}
		}
	}

	if err := nodeStore.Add(msg); err != nil {
		klog.Errorf("fail to add message %v nodeStore, err: %v", msg, err)
		return
	}
	nodeQueue.Add(messageKey)
}

func getMsgKey(obj interface{}) (string, error) {
	msg := obj.(*beehiveModel.Message)

	if msg.GetGroup() == edgeconst.GroupResource {
		return common.GetMessageUID(*msg)
	}

	return "", fmt.Errorf("failed to get message key")
}

func getListMsgKey(obj interface{}) (string, error) {
	msg := obj.(*beehiveModel.Message)

	return msg.Header.ID, nil
}

func isListResource(msg *beehiveModel.Message) bool {
	msgResource := msg.GetResource()
	if strings.Contains(msgResource, beehiveModel.ResourceTypePodlist) ||
		strings.Contains(msgResource, "membership") ||
		strings.Contains(msgResource, "twin/cloud_updated") ||
		strings.Contains(msgResource, beehiveModel.ResourceTypeServiceAccountToken) ||
		isVolumeOperation(msg.GetOperation()) {
		return true
	}

	if msg.Router.Operation == application.ApplicationResp {
		return true
	}
	if msg.GetOperation() == beehiveModel.ResponseOperation {
		content, ok := msg.Content.(string)
		if ok && content == commonconst.MessageSuccessfulContent {
			return true
		}
	}

	if msg.GetSource() == modules.EdgeControllerModuleName {
		resourceType, _ := messagelayer.GetResourceType(*msg)
		if resourceType == beehiveModel.ResourceTypeNode || resourceType == beehiveModel.ResourceTypeLease {
			return true
		}
	}
	// user data
	if msg.GetGroup() == modules.UserGroup {
		return true
	}

	return false
}

func isVolumeOperation(op string) bool {
	return op == commonconst.CSIOperationTypeCreateVolume ||
		op == commonconst.CSIOperationTypeDeleteVolume ||
		op == commonconst.CSIOperationTypeControllerPublishVolume ||
		op == commonconst.CSIOperationTypeControllerUnpublishVolume
}

func isDeleteMessage(msg *beehiveModel.Message) bool {
	if msg.GetOperation() == beehiveModel.DeleteOperation {
		return true
	}
	deletionTimestamp, err := common.GetMessageDeletionTimestamp(msg)
	if err != nil {
		klog.Errorf("fail to get message DeletionTimestamp for message: %s", msg.Header.ID)
		return false
	} else if deletionTimestamp != nil {
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

	return "", fmt.Errorf("no nodeID in Message.Router.Resource: %s", resource)
}

// Connect allocates the queues and stores for given node
func (q *ChannelMessageQueue) Connect(info *model.HubInfo) {
	_, queueExist := q.queuePool.Load(info.NodeID)
	_, storeExit := q.storePool.Load(info.NodeID)

	_, listQueueExist := q.listQueuePool.Load(info.NodeID)
	_, listStoreExit := q.listStorePool.Load(info.NodeID)

	if queueExist && storeExit && listQueueExist && listStoreExit {
		klog.Infof("Message queue and store for edge node %s are already exist", info.NodeID)
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
		klog.Warningf("Channel for edge node %s is already removed", info.NodeID)
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

// Publish sends message via the channel to Controllers
func (q *ChannelMessageQueue) Publish(msg *beehiveModel.Message) error {
	switch msg.Router.Source {
	case application.MetaServerSource:
		beehiveContext.Send(modules.DynamicControllerModuleName, *msg)
	case model.ResTwin:
		beehiveContext.SendToGroup(modules.DeviceControllerModuleGroup, *msg)
	default:
		beehiveContext.SendToGroup(modules.EdgeControllerGroupName, *msg)
	}
	return nil
}

// GetNodeQueue returns the queue for given node
func (q *ChannelMessageQueue) GetNodeQueue(nodeID string) workqueue.RateLimitingInterface {
	queue, ok := q.queuePool.Load(nodeID)
	if !ok {
		klog.Warningf("nodeQueue for edge node %s not found and created now", nodeID)
		nodeQueue := workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), nodeID)
		q.queuePool.Store(nodeID, nodeQueue)
		return nodeQueue
	}

	nodeQueue := queue.(workqueue.RateLimitingInterface)
	return nodeQueue
}

// GetNodeListQueue returns the listQueue for given node
func (q *ChannelMessageQueue) GetNodeListQueue(nodeID string) workqueue.RateLimitingInterface {
	queue, ok := q.listQueuePool.Load(nodeID)
	if !ok {
		klog.Warningf("nodeListQueue for edge node %s not found and created now", nodeID)
		nodeListQueue := workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), nodeID)
		q.listQueuePool.Store(nodeID, nodeListQueue)
		return nodeListQueue
	}

	nodeListQueue := queue.(workqueue.RateLimitingInterface)
	return nodeListQueue
}

// GetNodeStore returns the store for given node
func (q *ChannelMessageQueue) GetNodeStore(nodeID string) cache.Store {
	store, ok := q.storePool.Load(nodeID)
	if !ok {
		klog.Warningf("nodeStore for edge node %s not found and created now", nodeID)
		nodeStore := cache.NewStore(getMsgKey)
		q.storePool.Store(nodeID, nodeStore)
		return nodeStore
	}

	nodeStore := store.(cache.Store)
	return nodeStore
}

// GetNodeListStore returns the listStore for given node
func (q *ChannelMessageQueue) GetNodeListStore(nodeID string) cache.Store {
	store, ok := q.listStorePool.Load(nodeID)
	if !ok {
		klog.Warningf("nodeListStore for edge node %s not found and created now", nodeID)
		nodeListStore := cache.NewStore(getListMsgKey)
		q.listStorePool.Store(nodeID, nodeListStore)
		return nodeListStore
	}

	nodeListStore := store.(cache.Store)
	return nodeListStore
}
