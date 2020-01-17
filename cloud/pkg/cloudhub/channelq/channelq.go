package channelq

import (
	"fmt"
	"strings"
	"sync"

	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"
	"k8s.io/klog"

	beehiveContext "github.com/kubeedge/beehive/pkg/core/context"
	beehiveModel "github.com/kubeedge/beehive/pkg/core/model"
	"github.com/kubeedge/kubeedge/cloud/pkg/cloudhub/common/model"
	deviceconst "github.com/kubeedge/kubeedge/cloud/pkg/devicecontroller/constants"
	edgeconst "github.com/kubeedge/kubeedge/cloud/pkg/edgecontroller/constants"
	edgemessagelayer "github.com/kubeedge/kubeedge/cloud/pkg/edgecontroller/messagelayer"
	"github.com/kubeedge/kubeedge/common/constants"
)

// ChannelMessageQueue is the channel implementation of MessageQueue
type ChannelMessageQueue struct {
	queuePool sync.Map
	storePool sync.Map
}

// NewChannelMessageQueue initializes a new ChannelMessageQueue
func NewChannelMessageQueue() *ChannelMessageQueue {
	return &ChannelMessageQueue{}
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
		nodeID, err := GetNodeID(msg)
		if nodeID == "" || err != nil {
			klog.Warning("node id is not found in the message")
			continue
		}

		source, err := GetSource(msg)

		if source == "" || err != nil {
			klog.Warning("source is not found in the message")
			continue
		}

		nodeQueue, err := q.GetNodeQueue(nodeID)
		nodeStore, err := q.GetNodeStore(nodeID)

		if err != nil {
			klog.Infof("fail to get dispatch Node Queue for Node: %s, Source : %s", nodeID, source)
			continue
		}

		key, _ := getMsgKey(&msg)
		nodeQueue.Add(key)
		nodeStore.Add(&msg)
	}
}

func getMsgKey(obj interface{}) (string, error) {
	msg := obj.(*beehiveModel.Message)

	if msg.GetGroup() == edgeconst.GroupResource {
		resourceType, _ := edgemessagelayer.GetResourceType(*msg)
		resourceNamespace, _ := edgemessagelayer.GetNamespace(*msg)
		resourceName, _ := edgemessagelayer.GetResourceName(*msg)
		return resourceType + "/" + resourceNamespace + "/" + resourceName, nil
	}
	if msg.GetGroup() == deviceconst.GroupTwin {
		sli := strings.Split(msg.GetResource(), constants.ResourceSep)
		resourceType := sli[len(sli)-2]
		resourceName := sli[len(sli)-1]
		return resourceType + "/" + resourceName, nil
	}
	return "", fmt.Errorf("")
}

// getNodeID from "beehive/pkg/core/model".Message.Router.Resource
func GetNodeID(msg beehiveModel.Message) (string, error) {
	resource := msg.Router.Resource
	tokens := strings.Split(resource, constants.ResourceSep)
	numOfTokens := len(tokens)
	for i, token := range tokens {
		if token == model.ResNode && i+1 < numOfTokens && tokens[i+1] != "" {
			return tokens[i+1], nil
		}
	}

	return "", fmt.Errorf("No nodeId in Message.Router.Resource: %s", resource)
}

// getSource from "beehive/pkg/core/model".Message.Router.Source
func GetSource(msg beehiveModel.Message) (string, error) {
	source := msg.Router.Source
	return source, nil
}

// Connect allocates rChannel for given project and group
func (q *ChannelMessageQueue) Connect(info *model.HubInfo) error {
	_, queueExist := q.queuePool.Load(info.NodeID)
	_, storeExit := q.storePool.Load(info.NodeID)

	if queueExist && storeExit {
		return fmt.Errorf("edge node %s is already connected", info.NodeID)
	}

	queue := workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), info.NodeID)
	store := cache.NewStore(getMsgKey)

	_, ok := q.queuePool.LoadOrStore(info.NodeID, queue)
	_, ok = q.storePool.LoadOrStore(info.NodeID, store)

	if ok {
		// rchannel is already allocated
		return fmt.Errorf("edge node %s is already connected", info.NodeID)
	}

	return nil
}

// Close closes queues for given node
func (q *ChannelMessageQueue) Close(info *model.HubInfo) error {
	_, queueExist := q.queuePool.Load(info.NodeID)
	_, storeExist := q.storePool.Load(info.NodeID)

	if !queueExist && !storeExist {
		klog.Warningf("rChannel for edge node %s is already removed", info.NodeID)
		return nil
	}

	if queueExist {
		q.queuePool.Delete(info.NodeID)
	}
	if storeExist {
		q.storePool.Delete(info.NodeID)
	}

	return nil
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
		klog.Errorf("nodeQueue for edge node %s is removed", nodeID)
		return nil, fmt.Errorf("nodeQueue for edge node %s not found", nodeID)
	}

	nodeQueue := queue.(workqueue.RateLimitingInterface)
	return nodeQueue, nil
}

func (q *ChannelMessageQueue) GetNodeStore(nodeID string) (cache.Store, error) {
	store, ok := q.storePool.Load(nodeID)

	if !ok {
		klog.Errorf("nodeStore for edge node %s is removed", nodeID)
		return nil, fmt.Errorf("nodeStore for edge node %s not found", nodeID)
	}

	nodeStore := store.(cache.Store)
	return nodeStore, nil
}
