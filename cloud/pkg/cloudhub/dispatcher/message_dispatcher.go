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

package dispatcher

import (
	"context"
	"fmt"
	"strings"
	"sync"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/klog/v2"

	beehivecontext "github.com/kubeedge/beehive/pkg/core/context"
	beehivemodel "github.com/kubeedge/beehive/pkg/core/model"
	"github.com/kubeedge/kubeedge/cloud/pkg/cloudhub/common"
	"github.com/kubeedge/kubeedge/cloud/pkg/cloudhub/common/model"
	"github.com/kubeedge/kubeedge/cloud/pkg/cloudhub/session"
	"github.com/kubeedge/kubeedge/cloud/pkg/common/messagelayer"
	"github.com/kubeedge/kubeedge/cloud/pkg/common/modules"
	"github.com/kubeedge/kubeedge/cloud/pkg/synccontroller"
	commonconst "github.com/kubeedge/kubeedge/common/constants"
	"github.com/kubeedge/kubeedge/pkg/apis/reliablesyncs/v1alpha1"
	reliableclient "github.com/kubeedge/kubeedge/pkg/client/clientset/versioned"
	synclisters "github.com/kubeedge/kubeedge/pkg/client/listers/reliablesyncs/v1alpha1"
	"github.com/kubeedge/kubeedge/pkg/metaserver"
	"github.com/kubeedge/kubeedge/pkg/metaserver/util"
)

// There are two `AcknowledgeMode` for message that send to edge node
// ------------------------------------------------------------------
// ACK mode: In this mode, the edge node MUST send acknowledgement to
// the CloudHub for the messages it receive after successfully process.
// After metaManger successfully stores the message in the edge node,
// an acknowledgement will be sent to the CloudHub to inform that the
// message was successfully processed. If the connection lost before edge
// node sends acknowledgement for the message, the server will assume the
// message has not been processed successfully and will resend the message
// to the edge node until CloudHub receive the acknowledgement message.
// ------------------------------------------------------------------
// NO-ACK mode: In this mode, edge node does not send acknowledgement
// to the CloudHub for the messages it receives. The CloudHub will
// assume the edge node has received the message and is successfully
// processed. This acknowledgment mode can cause messages being transmitted
// to the edge node to get dropped. But this mode usually is used when
// send response message for request from the edge, so if the edge node
// do not receive the message, it will issue a new request and try again.
// ------------------------------------------------------------------

// MessageDispatcher is responsible for the dispatch of upstream messages
// (edge ​​to cloud) and downstream messages (cloud to edge)
type MessageDispatcher interface {
	// DispatchDownstream continuously reads the messages from cloudHub module,
	// and according to the content of the message, the message is dispatched
	// to the message queue of each edge node.
	DispatchDownstream()

	// DispatchUpstream dispatch messages sent from edge nodes to the cloud,
	// such as node status messages, pod status messages, etc.
	DispatchUpstream(message *beehivemodel.Message, info *model.HubInfo)

	// AddNodeMessagePool add the given node message pool to the dispatcher.
	AddNodeMessagePool(nodeID string, pool *common.NodeMessagePool)

	// DeleteNodeMessagePool deletes the given node message pool from the dispatcher.
	DeleteNodeMessagePool(nodeID string, pool *common.NodeMessagePool)

	// GetNodeMessagePool provides the nodeMessagePool that matches node ID
	GetNodeMessagePool(nodeID string) *common.NodeMessagePool

	// Publish sends the given message to module according to the message source
	Publish(msg *beehivemodel.Message) error
}

type messageDispatcher struct {
	// NodeMessagePools stores and manages access to nodeMessagePool, maintaining
	// the mappings between nodeID and its nodeMessagePool.
	NodeMessagePools sync.Map

	// SessionManager
	SessionManager *session.Manager

	// objectSync client for interacting with Kubernetes API servers.
	reliableClient reliableclient.Interface

	// objectSyncLister can list/get objectSync from the shared informer's store
	objectSyncLister synclisters.ObjectSyncLister

	// clusterObjectSyncLister can list/get clusterObjectSync from the shared informer's store
	clusterObjectSyncLister synclisters.ClusterObjectSyncLister
}

// NewMessageDispatcher initializes a new MessageDispatcher
func NewMessageDispatcher(
	sessionManager *session.Manager,
	objectSyncLister synclisters.ObjectSyncLister,
	clusterObjectSyncLister synclisters.ClusterObjectSyncLister,
	reliableClient reliableclient.Interface) MessageDispatcher {
	return &messageDispatcher{
		objectSyncLister:        objectSyncLister,
		clusterObjectSyncLister: clusterObjectSyncLister,
		reliableClient:          reliableClient,
		SessionManager:          sessionManager,
	}
}

func (md *messageDispatcher) DispatchDownstream() {
	for {
		select {
		case <-beehivecontext.Done():
			klog.Warning("DispatchDownstream loop stopped")
			return

		default:
			msg, err := beehivecontext.Receive(model.SrcCloudHub)
			if err != nil {
				klog.Errorf("receive message failed %v", err)
				continue
			}

			klog.V(4).Infof("[DispatchDownstream] dispatch Message to edge: %+v", msg)

			nodeID, err := GetNodeID(&msg)
			if nodeID == "" || err != nil {
				klog.Warningf("node id is not found in the message: %+v", msg)
				continue
			}

			if !model.IsToEdge(&msg) {
				klog.Warningf("skip message not to edge node %s: %+v, content %s", nodeID, msg)
				continue
			}

			switch {
			case noAckRequired(&msg):
				md.enqueueNoAckMessage(nodeID, &msg)
			default:
				md.enqueueAckMessage(nodeID, &msg)
			}
		}
	}
}

func (md *messageDispatcher) DispatchUpstream(message *beehivemodel.Message, info *model.HubInfo) {
	switch {
	case message.GetOperation() == model.OpKeepalive:
		klog.V(4).Infof("Keepalive message received from node: %s", info.NodeID)

		err := md.SessionManager.KeepAliveMessage(info.NodeID)
		if err != nil {
			klog.Errorf("node %s receive keep alive message err: %v", info.NodeID, err)
		}

	case common.IsVolumeResource(message.GetResource()):
		beehivecontext.SendResp(*message)

	case message.Router.Operation == beehivemodel.ResponseOperation:
		err := md.SessionManager.ReceiveMessageAck(info.NodeID, message.Header.ParentID)
		if err != nil {
			klog.Errorf("node %s receive message ack err: %v", info.NodeID, err)
		}

	case message.GetOperation() == beehivemodel.UploadOperation && message.GetGroup() == modules.UserGroup:
		message.Router.Resource = fmt.Sprintf("node/%s/%s", info.NodeID, message.Router.Resource)
		beehivecontext.Send(modules.RouterModuleName, *message)

	default:
		err := md.PubToController(info, message)
		if err != nil {
			// content is not logged since it may contain sensitive information
			klog.Errorf("Failed PubToController nodeID %s, message: %s, error: %v", info.NodeID, message.String(), err)
		}
	}
}

func (md *messageDispatcher) PubToController(info *model.HubInfo, msg *beehivemodel.Message) error {
	msg.SetResourceOperation(fmt.Sprintf("node/%s/%s", info.NodeID, msg.GetResource()), msg.GetOperation())
	if model.IsFromEdge(msg) {
		return md.Publish(msg)
	}
	return nil
}

func (md *messageDispatcher) enqueueNoAckMessage(nodeID string, msg *beehivemodel.Message) {
	nodeMessagePool := md.GetNodeMessagePool(nodeID)

	messageKey, _ := common.NoAckMessageKeyFunc(msg)
	if err := nodeMessagePool.NoAckMessageStore.Add(msg); err != nil {
		klog.Errorf("failed to add msg: %v", err)
		return
	}
	nodeMessagePool.NoAckMessageQueue.Add(messageKey)
}

func (md *messageDispatcher) enqueueAckMessage(nodeID string, msg *beehivemodel.Message) {
	// Message that require ack MUST have resource version.
	if msg.GetResourceVersion() == "" && !isDeleteMessage(msg) {
		return
	}

	nodeMessagePool := md.GetNodeMessagePool(nodeID)
	nodeQueue := nodeMessagePool.AckMessageQueue
	nodeStore := nodeMessagePool.AckMessageStore

	messageKey, err := common.AckMessageKeyFunc(msg)
	if err != nil {
		klog.Errorf("fail to get key for message: %s", msg.String())
		return
	}

	shouldEnqueue := false
	defer func() {
		if shouldEnqueue {
			if err := nodeStore.Add(msg); err != nil {
				klog.Errorf("fail to add message %v nodeStore, err: %v", msg, err)
				return
			}
			nodeQueue.Add(messageKey)
		}
	}()

	// If the message operation is delete, force to sync the resource message
	// If the message operation is response, force to sync the resource message,
	// since the edgeCore requests it.
	if isDeleteMessage(msg) || msg.GetOperation() == beehivemodel.ResponseOperation {
		shouldEnqueue = true
		return
	}

	item, exist, _ := nodeStore.GetByKey(messageKey)
	if exist {
		msgInStore := item.(*beehivemodel.Message)

		if isDeleteMessage(msgInStore) ||
			synccontroller.CompareResourceVersion(msg.GetResourceVersion(), msgInStore.GetResourceVersion()) <= 0 {
			// If the message resource version is older than the message in store or the operation
			// for the message in store is delete, The message will be discarded directly.
			return
		}
		shouldEnqueue = true
		return
	}

	// If the message doesn't exist in the store, then compare it with the version stored in the objectSync.
	resourceNamespace, _ := messagelayer.GetNamespace(*msg)
	resourceName, _ := messagelayer.GetResourceName(*msg)
	resourceUID, err := common.GetMessageUID(*msg)
	if err != nil {
		klog.Errorf("fail to get message UID for message: %s", msg.Header.ID)
		return
	}

	objectSyncName := synccontroller.BuildObjectSyncName(nodeID, resourceUID)
	objectSync, err := md.objectSyncLister.ObjectSyncs(resourceNamespace).Get(objectSyncName)

	switch {
	case err == nil && objectSync.Status.ObjectResourceVersion != "":
		if synccontroller.CompareResourceVersion(msg.GetResourceVersion(), objectSync.Status.ObjectResourceVersion) > 0 {
			shouldEnqueue = true
			return
		}

	case err != nil && apierrors.IsNotFound(err):
		// If objectSync is not exist, this indicates that the message is coming
		// for the first time, We create objectSync for the resource directly.
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

		objectSyncStatus, err := md.reliableClient.
			ReliablesyncsV1alpha1().
			ObjectSyncs(resourceNamespace).
			Create(context.Background(), objectSync, metav1.CreateOptions{})
		if err != nil {
			klog.ErrorS(err, "Failed to create objectSync",
				"objectSyncName", objectSyncName,
				"resourceNamespace", resourceNamespace,
				"resourceName", resourceName)
			return
		}

		objectSyncStatus.Status.ObjectResourceVersion = "0"
		_, err = md.reliableClient.
			ReliablesyncsV1alpha1().
			ObjectSyncs(resourceNamespace).
			UpdateStatus(context.Background(), objectSyncStatus, metav1.UpdateOptions{})
		if err != nil {
			klog.ErrorS(err, "Failed to update objectSync",
				"objectSyncName", objectSyncName,
				"resourceNamespace", resourceNamespace,
				"resourceName", resourceName)
		}

		// enqueue message that comes for the first time
		shouldEnqueue = true

	case err != nil:
		klog.Errorf("failed to get ObjectSync %s/%s: %v", resourceNamespace, objectSyncName, err)
	}
}

func isDeleteMessage(msg *beehivemodel.Message) bool {
	if msg.GetOperation() == beehivemodel.DeleteOperation {
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
func GetNodeID(msg *beehivemodel.Message) (string, error) {
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

func noAckRequired(msg *beehivemodel.Message) bool {
	msgResource := msg.GetResource()
	switch {
	case strings.Contains(msgResource, beehivemodel.ResourceTypePodlist):
		return true
	case strings.Contains(msgResource, "membership"):
		return true
	case strings.Contains(msgResource, "twin/cloud_updated"):
		return true
	case strings.Contains(msgResource, beehivemodel.ResourceTypeServiceAccountToken):
		return true
	case isVolumeOperation(msg.GetOperation()):
		return true
	case msg.Router.Operation == metaserver.ApplicationResp:
		return true
	case msg.GetGroup() == modules.UserGroup:
		return true
	case msg.GetSource() == modules.NodeUpgradeJobControllerModuleName:
		return true
	case msg.GetOperation() == beehivemodel.ResponseOperation:
		content, ok := msg.Content.(string)
		if ok && content == commonconst.MessageSuccessfulContent {
			return true
		}
		fallthrough
	default:
		if msg.GetSource() == modules.EdgeControllerModuleName {
			resourceType, _ := messagelayer.GetResourceType(*msg)
			if resourceType == beehivemodel.ResourceTypeNode ||
				resourceType == beehivemodel.ResourceTypeLease ||
				resourceType == beehivemodel.ResourceTypeNodePatch ||
				resourceType == beehivemodel.ResourceTypePodPatch ||
				resourceType == beehivemodel.ResourceTypePodStatus {
				return true
			}
		}
	}
	return false
}

func isVolumeOperation(op string) bool {
	return op == commonconst.CSIOperationTypeCreateVolume ||
		op == commonconst.CSIOperationTypeDeleteVolume ||
		op == commonconst.CSIOperationTypeControllerPublishVolume ||
		op == commonconst.CSIOperationTypeControllerUnpublishVolume
}

// GetNodeMessagePool returns the message pool for given node
func (md *messageDispatcher) GetNodeMessagePool(nodeID string) *common.NodeMessagePool {
	nsp, exist := md.NodeMessagePools.Load(nodeID)
	if !exist {
		klog.Warningf("message pool for edge node %s not found and created now", nodeID)
		nodeMessagePool := common.InitNodeMessagePool(nodeID)
		md.NodeMessagePools.Store(nodeID, nodeMessagePool)
		return nodeMessagePool
	}

	return nsp.(*common.NodeMessagePool)
}

func (md *messageDispatcher) AddNodeMessagePool(nodeID string, pool *common.NodeMessagePool) {
	md.NodeMessagePools.Store(nodeID, pool)
}

func (md *messageDispatcher) DeleteNodeMessagePool(nodeID string, pool *common.NodeMessagePool) {
	nsp, exist := md.NodeMessagePools.Load(nodeID)
	if !exist {
		klog.Warningf("message pool not found for node %s", nodeID)
		return
	}

	// This usually happens when the node is disconnect then quickly reconnect
	if nsp.(*common.NodeMessagePool) != pool {
		klog.Warningf("the message pool %s already deleted", nodeID)
		return
	}

	md.NodeMessagePools.Delete(nodeID)
}

func (md *messageDispatcher) Publish(msg *beehivemodel.Message) error {
	switch msg.Router.Source {
	case metaserver.MetaServerSource:
		beehivecontext.Send(modules.DynamicControllerModuleName, *msg)
	case model.ResTwin:
		beehivecontext.SendToGroup(modules.DeviceControllerModuleGroup, *msg)
	default:
		beehivecontext.SendToGroup(modules.EdgeControllerGroupName, *msg)
	}
	return nil
}
