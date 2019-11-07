package channelq

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"k8s.io/klog"

	beehiveContext "github.com/kubeedge/beehive/pkg/core/context"
	beehiveModel "github.com/kubeedge/beehive/pkg/core/model"
	"github.com/kubeedge/kubeedge/cloud/pkg/cloudhub/common/model"
)

// Read channel buffer size
const (
	rChanBufSize = 10
)

// MessageSet holds a set of messages
type MessageSet interface {
	Ack() error
	Get() (*beehiveModel.Message, error)
}

// ChannelMessageSet is the channel implementation of MessageSet
type ChannelMessageSet struct {
	current  beehiveModel.Message
	messages <-chan beehiveModel.Message
}

// NewChannelMessageSet initializes a new ChannelMessageSet instance
func NewChannelMessageSet(messages <-chan beehiveModel.Message) *ChannelMessageSet {
	return &ChannelMessageSet{messages: messages}
}

// Ack acknowledges once the event is processed
func (s *ChannelMessageSet) Ack() error {
	return nil
}

// Get obtains one event from the queue
func (s *ChannelMessageSet) Get() (*beehiveModel.Message, error) {
	var ok bool
	s.current, ok = <-s.messages
	if !ok {
		return nil, fmt.Errorf("failed to get message from cluster, reason: channel is closed")
	}
	return &s.current, nil
}

// ChannelMessageQueue is the channel implementation of MessageQueue
type ChannelMessageQueue struct {
	ctx         *beehiveContext.Context
	channelPool sync.Map
}

// NewChannelMessageQueue initializes a new ChannelMessageQueue
func NewChannelMessageQueue(ctx *beehiveContext.Context) *ChannelMessageQueue {
	q := ChannelMessageQueue{ctx: ctx}
	return &q
}

// DispatchMessage gets the message from the cloud, extracts the
// node id from it, gets the channel associated with the node
// and pushes the event on the channel
func (q *ChannelMessageQueue) DispatchMessage(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			klog.Warningf("Cloudhub channel eventqueue dispatch message loop stoped")
			return
		default:
		}
		msg, err := q.ctx.Receive(model.SrcCloudHub)
		if err != nil {
			klog.Info("receive not Message format message")
			continue
		}
		resource := msg.Router.Resource
		tokens := strings.Split(resource, "/")
		numOfTokens := len(tokens)
		var nodeID string
		for i, token := range tokens {
			if token == model.ResNode && i+1 < numOfTokens {
				nodeID = tokens[i+1]
				break
			}
		}
		if nodeID == "" {
			klog.Warning("node id is not found in the message")
			continue
		}
		rChannel, err := q.getRChannel(nodeID)
		if err != nil {
			klog.Infof("fail to get dispatch channel for %s", nodeID)
			continue
		}
		rChannel <- msg
	}
}

func (q *ChannelMessageQueue) getRChannel(nodeID string) (chan beehiveModel.Message, error) {
	channels, ok := q.channelPool.Load(nodeID)
	if !ok {
		klog.Errorf("rChannel for edge node %s is removed", nodeID)
		return nil, fmt.Errorf("rChannel not found")
	}
	rChannel := channels.(chan beehiveModel.Message)
	return rChannel, nil
}

// Connect allocates rChannel for given project and group
func (q *ChannelMessageQueue) Connect(info *model.HubInfo) error {
	_, ok := q.channelPool.Load(info.NodeID)
	if ok {
		return fmt.Errorf("edge node %s is already connected", info.NodeID)
	}
	// allocate a new rchannel with default buffer size
	rChannel := make(chan beehiveModel.Message, rChanBufSize)
	_, ok = q.channelPool.LoadOrStore(info.NodeID, rChannel)
	if ok {
		// rchannel is already allocated
		return fmt.Errorf("edge node %s is already connected", info.NodeID)
	}
	return nil
}

// Close closes rChannel for given project and group
func (q *ChannelMessageQueue) Close(info *model.HubInfo) error {
	channels, ok := q.channelPool.Load(info.NodeID)
	if !ok {
		klog.Warningf("rChannel for edge node %s is already removed", info.NodeID)
		return nil
	}
	rChannel := channels.(chan beehiveModel.Message)
	close(rChannel)
	q.channelPool.Delete(info.NodeID)
	return nil
}

// Publish sends message via the rchannel to Edge Controller
func (q *ChannelMessageQueue) Publish(msg *beehiveModel.Message) error {
	switch msg.Router.Source {
	case model.ResTwin:
		q.ctx.SendToGroup(model.SrcDeviceController, *msg)
	default:
		q.ctx.SendToGroup(model.SrcEdgeController, *msg)
	}
	return nil
}

// Consume retrieves message from the rChannel for given project and group
func (q *ChannelMessageQueue) Consume(info *model.HubInfo) (MessageSet, error) {
	rChannel, err := q.getRChannel(info.NodeID)
	if err != nil {
		return nil, err
	}
	return NewChannelMessageSet((<-chan beehiveModel.Message)(rChannel)), nil
}

// Workload returns the number of queue channels connected to queue
func (q *ChannelMessageQueue) Workload() (float64, error) {
	return 1, nil
}
