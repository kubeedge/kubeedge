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

package session

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/klog/v2"

	beehivemodel "github.com/kubeedge/beehive/pkg/core/model"
	"github.com/kubeedge/kubeedge/cloud/pkg/cloudhub/common"
	"github.com/kubeedge/kubeedge/cloud/pkg/cloudhub/common/model"
	"github.com/kubeedge/kubeedge/cloud/pkg/common/messagelayer"
	deviceconst "github.com/kubeedge/kubeedge/cloud/pkg/devicecontroller/constants"
	edgeconst "github.com/kubeedge/kubeedge/cloud/pkg/edgecontroller/constants"
	"github.com/kubeedge/kubeedge/cloud/pkg/synccontroller"
	"github.com/kubeedge/kubeedge/pkg/apis/reliablesyncs/v1alpha1"
	reliableclient "github.com/kubeedge/kubeedge/pkg/client/clientset/versioned"
	"github.com/kubeedge/kubeedge/pkg/metaserver/util"
	"github.com/kubeedge/viaduct/pkg/conn"
)

var sendRetryInterval = 5 * time.Second

// session termination error type
const (
	NoErr = iota
	TransportErr
	NodeStopErr
	QueueShutdownErr
)

// ErrWaitTimeout is returned when the condition exited without success.
var ErrWaitTimeout = errors.New("timed out waiting for the condition")

// NodeSession is an abstraction of a node connection lifecycle.
type NodeSession struct {
	// nodeID is the identifier of the edge node and is unique in the cluster
	nodeID string

	// projectID is the project ID to which the edge node belongs
	projectID string

	// connection is the underlying net connection (websocket or QUIC)
	connection conn.Connection

	// keepaliveInterval is the interval in seconds that keepalive messages
	// are received from the peer.
	keepaliveInterval time.Duration

	// keepaliveChan defines a chan which will receive the keepalive message
	keepaliveChan chan struct{}

	// nodeMessagePool stores all the message that will send to an single edge node
	nodeMessagePool *common.NodeMessagePool

	// ackMessageCache records the mapping of message ID to its response channel
	ackMessageCache sync.Map

	// reliableClient the objectSync client for interacting with Kubernetes API servers
	reliableClient reliableclient.Interface

	// terminateErr records the error type of session termination
	terminateErr int32

	// stopOnce is used to mark that session Terminating can only be executed once
	stopOnce sync.Once

	ctx        context.Context
	cancelFunc context.CancelFunc
}

func NewNodeSession(
	nodeID, projectID string,
	connection conn.Connection,
	keepaliveInterval time.Duration,
	nodeMessagePool *common.NodeMessagePool,
	reliableClient reliableclient.Interface,
) *NodeSession {
	ctx, cancelFunc := context.WithCancel(context.Background())
	return &NodeSession{
		ctx:               ctx,
		cancelFunc:        cancelFunc,
		nodeID:            nodeID,
		projectID:         projectID,
		connection:        connection,
		keepaliveInterval: keepaliveInterval,
		keepaliveChan:     make(chan struct{}, 1),
		nodeMessagePool:   nodeMessagePool,
		reliableClient:    reliableClient,
		terminateErr:      NoErr,
	}
}

// KeepAliveMessage receive keepalive message from edge node
func (ns *NodeSession) KeepAliveMessage() {
	select {
	case ns.keepaliveChan <- struct{}{}:
	default:
		klog.Warningf("keepaliveChan is full for %s", ns.nodeID)
	}
}

// ReceiveMessageAck receive the message ack from edge node
func (ns *NodeSession) ReceiveMessageAck(parentID string) {
	ackChan, exist := ns.ackMessageCache.Load(parentID)
	if exist {
		close(ackChan.(chan struct{}))
		ns.ackMessageCache.Delete(parentID)
	}
}

// Start the main goroutine responsible for serving node session
func (ns *NodeSession) Start() {
	klog.Infof("Start session for edge node %s", ns.nodeID)

	go ns.KeepAliveCheck()
	go ns.SendAckMessage()
	go ns.SendNoAckMessage()

	<-ns.ctx.Done()
}

// KeepAliveCheck
// A goroutine running KeepAliveCheck is started for each connection.
func (ns *NodeSession) KeepAliveCheck() {
	keepaliveTimer := time.NewTimer(ns.keepaliveInterval)

	for {
		// timer may be not active, and fired
		if !keepaliveTimer.Stop() {
			select {
			case <-keepaliveTimer.C:
			default:
			}
		}

		keepaliveTimer.Reset(ns.keepaliveInterval)

		select {
		case <-ns.ctx.Done():
			return

		case _, ok := <-ns.keepaliveChan:
			if !ok {
				klog.Errorf("Stop keepalive check for node: %s", ns.nodeID)
				return
			}
			klog.V(4).Infof("Node %s is still alive", ns.nodeID)

		case <-keepaliveTimer.C:
			klog.Errorf("timeout to receive keepalive for node %s", ns.nodeID)

			ns.SetTerminateErr(TransportErr)

			// Terminating node session
			ns.Terminating()
		}
	}
}

// SendAckMessage loops forever sending message that require acknowledgment
// to the edge node until an error is encountered (or the connection is closed).
func (ns *NodeSession) SendAckMessage() {
	for {
		select {
		case <-ns.ctx.Done():
			return

		default:
			exit, err := ns.syncAckMessage()
			if err != nil {
				klog.Errorf("syncAckMessage err: %v", err)
			}

			if exit {
				ns.Terminating()

				// exit loop
				return
			}
		}
	}
}

// SendNoAckMessage loops forever sending the message that does not require acknowledgment
// to the edge node until an error is encountered (or the connection is closed).
func (ns *NodeSession) SendNoAckMessage() {
	for {
		select {
		case <-ns.ctx.Done():
			return

		default:
			exit, err := ns.syncNoAckMessage()
			if err != nil {
				klog.Errorf("syncNoAckMessage err: %v", err)
			}

			if exit {
				ns.Terminating()

				// exit loop
				return
			}
		}
	}
}

// Terminating terminates the node session and it is called when the client goes offline,
// It will shutdown the message queue and the goroutine associated with it will exit.
func (ns *NodeSession) Terminating() {
	ns.stopOnce.Do(func() {
		ns.cancelFunc()

		ns.nodeMessagePool.ShutDown()

		// ignore close error
		_ = ns.connection.Close()
	})
}

func (ns *NodeSession) SetTerminateErr(terminateErr int32) {
	if atomic.LoadInt32(&ns.terminateErr) != NoErr {
		return
	}

	atomic.StoreInt32(&ns.terminateErr, terminateErr)
}

func (ns *NodeSession) GetTerminateErr() int32 {
	return atomic.LoadInt32(&ns.terminateErr)
}

func (ns *NodeSession) syncNoAckMessage() (bool, error) {
	key, quit := ns.nodeMessagePool.NoAckMessageQueue.Get()
	if quit {
		ns.SetTerminateErr(QueueShutdownErr)
		return true, fmt.Errorf("NoAckMessageQueue for node %s has shutdown", ns.nodeID)
	}

	defer func() {
		// NoAckMessage will be deleted no matter send success or failure
		ns.nodeMessagePool.NoAckMessageQueue.Forget(key)
		// You must call Done with item when you have finished processing it.
		ns.nodeMessagePool.NoAckMessageQueue.Done(key)
	}()

	msg, err := ns.nodeMessagePool.GetNoAckMessage(key.(string))
	if err != nil {
		return false, err
	}

	defer func() {
		// delete message from the store
		if err := ns.nodeMessagePool.NoAckMessageStore.Delete(msg); err != nil {
			klog.Errorf("failed to delete message from store, err: %v", err)
		}
	}()

	if model.IsNodeStopped(msg) {
		ns.SetTerminateErr(NodeStopErr)
		klog.Warningf("node %s is deleted, message for node will be cleaned up", ns.nodeID)
		return true, nil
	}

	klog.V(4).Infof("send message to node %s, %s, content %s", ns.nodeID, msg.String(), msg.Content)

	common.TrimMessage(msg)

	if err := ns.connection.WriteMessageAsync(msg); err != nil {
		ns.SetTerminateErr(TransportErr)
		return true, fmt.Errorf("send message to edge node %s err: %v", ns.nodeID, err)
	}

	return false, nil
}

func (ns *NodeSession) syncAckMessage() (bool, error) {
	key, quit := ns.nodeMessagePool.AckMessageQueue.Get()
	if quit {
		ns.SetTerminateErr(QueueShutdownErr)
		return true, fmt.Errorf("AckMessageQueue for node %s has shutdown", ns.nodeID)
	}
	defer ns.nodeMessagePool.AckMessageQueue.Done(key)

	msg, err := ns.nodeMessagePool.GetAckMessage(key.(string))
	if err != nil {
		return false, err
	}

	klog.V(4).Infof("send message to node %s, %s, content %s", ns.nodeID, msg.String(), msg.Content)

	copyMsg := common.DeepCopy(msg)
	common.TrimMessage(copyMsg)

	err = ns.sendMessageWithRetry(copyMsg, msg)
	switch {
	case err == nil:
		// no err, forget this key and return
		ns.nodeMessagePool.AckMessageQueue.Forget(key)
		return false, nil

	case err == ErrWaitTimeout:
		// if err is timeout err, we will add the message to queue again
		ns.nodeMessagePool.AckMessageQueue.AddRateLimited(key)
		return false, fmt.Errorf("send message to node %s err: %v, message: %s", ns.nodeID, err, msg.String())

	default:
		ns.SetTerminateErr(TransportErr)
		// if err is Transport Error, we will terminating node session
		return true, err
	}
}

func (ns *NodeSession) sendMessageWithRetry(copyMsg, msg *beehivemodel.Message) error {
	ackChan := make(chan struct{})
	ns.ackMessageCache.Store(copyMsg.GetID(), ackChan)

	// initialize retry count and timer for sending message
	retryCount := 0
	ticker := time.NewTimer(sendRetryInterval)

	err := ns.connection.WriteMessageAsync(copyMsg)
	if err != nil {
		return err
	}

	for {
		select {
		case <-ackChan:
			ns.saveSuccessPoint(msg)
			return nil

		case <-ticker.C:
			if retryCount == 4 {
				return ErrWaitTimeout
			}

			err := ns.connection.WriteMessageAsync(copyMsg)
			if err != nil {
				return err
			}

			retryCount++
			ticker.Reset(sendRetryInterval)
		}
	}
}

func (ns *NodeSession) saveSuccessPoint(msg *beehivemodel.Message) {
	switch {
	case msg.GetGroup() == deviceconst.GroupTwin:
		// TODO: save device info
		return

	case msg.GetGroup() == edgeconst.GroupResource:
		resourceNamespace, _ := messagelayer.GetNamespace(*msg)
		resourceName, _ := messagelayer.GetResourceName(*msg)
		resourceType, _ := messagelayer.GetResourceType(*msg)

		resourceUID, err := common.GetMessageUID(*msg)
		if err != nil {
			klog.Errorf("failed to get message UID %v, err: %v", msg, err)
			return
		}

		objectSyncName := synccontroller.BuildObjectSyncName(ns.nodeID, resourceUID)

		if msg.GetOperation() == beehivemodel.DeleteOperation {
			ns.deleteSuccessPoint(resourceNamespace, objectSyncName, msg)
			return
		}

		objectSync, err := ns.reliableClient.
			ReliablesyncsV1alpha1().
			ObjectSyncs(resourceNamespace).
			Get(context.Background(), objectSyncName, metav1.GetOptions{})

		switch {
		case err == nil:
			objectSync.Status.ObjectResourceVersion = msg.GetResourceVersion()

			_, err := ns.reliableClient.
				ReliablesyncsV1alpha1().
				ObjectSyncs(resourceNamespace).
				UpdateStatus(context.Background(), objectSync, metav1.UpdateOptions{})

			if err != nil {
				klog.ErrorS(err, "failed to update objectSync",
					"objectSyncName", objectSyncName,
					"resourceType", resourceType,
					"resourceNamespace", resourceNamespace,
					"resourceName", resourceName)
				return
			}

		case apierrors.IsNotFound(err):
			// create objectSync if not found
			objectSync := &v1alpha1.ObjectSync{
				ObjectMeta: metav1.ObjectMeta{
					Name:      objectSyncName,
					Namespace: resourceNamespace,
				},
				Spec: v1alpha1.ObjectSyncSpec{
					ObjectAPIVersion: util.GetMessageAPIVersion(msg),
					ObjectKind:       util.GetMessageResourceType(msg),
					ObjectName:       resourceName,
				},
			}

			if objectSync.Spec.ObjectKind == "" {
				klog.ErrorS(nil, "failed to init objectSync, ObjectKind is empty",
					"objectSyncName", objectSyncName,
					"resourceType", resourceType,
					"resourceNamespace", resourceNamespace,
					"resourceName", resourceName,
					"message", msg.GetContent())
				return
			}

			objectSyncStatus, err := ns.reliableClient.
				ReliablesyncsV1alpha1().
				ObjectSyncs(resourceNamespace).
				Create(context.Background(), objectSync, metav1.CreateOptions{})
			if err != nil {
				klog.Errorf("Failed to create objectSync: %s, err: %v", objectSyncName, err)
				return
			}

			objectSyncStatus.Status.ObjectResourceVersion = msg.GetResourceVersion()
			_, err = ns.reliableClient.
				ReliablesyncsV1alpha1().
				ObjectSyncs(resourceNamespace).
				UpdateStatus(context.Background(), objectSyncStatus, metav1.UpdateOptions{})
			if err != nil {
				klog.Errorf("Failed to update objectSync: %s, err: %v", objectSyncName, err)
				return
			}

		default:
			// request objectSync from KubeAPIServer err
			klog.Errorf("Failed to get objectSync: %s, err: %v", objectSyncName, err)
			return
		}
	}

	klog.V(4).Infof("saveSuccessPoint successfully for message: %s", msg.GetResource())
}

func (ns *NodeSession) deleteSuccessPoint(resourceNamespace, objectSyncName string, msg *beehivemodel.Message) {
	err := ns.reliableClient.
		ReliablesyncsV1alpha1().
		ObjectSyncs(resourceNamespace).
		Delete(context.Background(), objectSyncName, *metav1.NewDeleteOptions(0))
	if err != nil && !apierrors.IsNotFound(err) {
		klog.Errorf("Delete ObjectSync %s error: %v", objectSyncName, err)
	}

	if err := ns.nodeMessagePool.AckMessageStore.Delete(msg); err != nil {
		klog.Errorf("failed to delete message %v, err: %v", msg, err)
	}
}
