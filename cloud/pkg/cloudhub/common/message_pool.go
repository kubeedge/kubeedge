/*
Copyright 2022 The KubeEdge Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

   http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package common

import (
	"fmt"

	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"

	beehivemodel "github.com/kubeedge/beehive/pkg/core/model"
)

// NodeMessagePool is a collection of all downstream messages sent to an
// edge node. There are two types of messages, one that requires an ack
// and one that does not. For each type of message, we use the `queue` to
// mark the order of sending, and use the `store` to store specific messages
type NodeMessagePool struct {
	// AckMessageStore store message that will send to edge node
	// and require acknowledgement from edge node.
	AckMessageStore cache.Store
	// AckMessageQueue store message key that will send to edge node
	// and require acknowledgement from edge node.
	AckMessageQueue workqueue.RateLimitingInterface
	// NoAckMessageStore store message that will send to edge node
	// and do not require acknowledgement from edge node.
	NoAckMessageStore cache.Store
	// NoAckMessageQueue store message key that will send to edge node
	// and do not require acknowledgement from edge node.
	NoAckMessageQueue workqueue.RateLimitingInterface
}

// InitNodeMessagePool init node message pool for node
func InitNodeMessagePool(nodeID string) *NodeMessagePool {
	return &NodeMessagePool{
		AckMessageStore:   cache.NewStore(AckMessageKeyFunc),
		AckMessageQueue:   workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), nodeID),
		NoAckMessageStore: cache.NewStore(NoAckMessageKeyFunc),
		NoAckMessageQueue: workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), nodeID),
	}
}

// GetAckMessage get message that requires ack with the key
func (nsp *NodeMessagePool) GetAckMessage(key string) (*beehivemodel.Message, error) {
	obj, exist, err := nsp.AckMessageStore.GetByKey(key)
	if err != nil {
		return nil, fmt.Errorf("get message %s err: %v", key, err)
	}

	if !exist {
		return nil, fmt.Errorf("message %s not found", key)
	}

	msg, ok := obj.(*beehivemodel.Message)
	if !ok {
		return nil, fmt.Errorf("message type %T is invalid", obj)
	}

	if msg == nil {
		return nil, fmt.Errorf("message is nil for key: %s", key)
	}

	return msg, nil
}

// GetNoAckMessage get message that does not require ack with the key
func (nsp *NodeMessagePool) GetNoAckMessage(key string) (*beehivemodel.Message, error) {
	obj, exist, err := nsp.NoAckMessageStore.GetByKey(key)
	if err != nil {
		return nil, fmt.Errorf("get message %s err: %v", key, err)
	}

	if !exist {
		return nil, fmt.Errorf("message %s not found", key)
	}

	msg, ok := obj.(*beehivemodel.Message)
	if !ok {
		return nil, fmt.Errorf("message type %T is invalid", obj)
	}

	if msg == nil {
		return nil, fmt.Errorf("message is nil for key: %s", key)
	}

	return msg, nil
}

// ShutDown will close all the message queue in the message pool
func (nsp *NodeMessagePool) ShutDown() {
	nsp.AckMessageQueue.ShutDown()
	nsp.NoAckMessageQueue.ShutDown()
}
