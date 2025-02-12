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

package handler

import (
	"context"
	"time"

	"github.com/avast/retry-go"
	"k8s.io/klog/v2"

	reliableclient "github.com/kubeedge/api/client/clientset/versioned"
	"github.com/kubeedge/kubeedge/cloud/pkg/cloudhub/authorization"
	"github.com/kubeedge/kubeedge/cloud/pkg/cloudhub/common"
	"github.com/kubeedge/kubeedge/cloud/pkg/cloudhub/common/model"
	"github.com/kubeedge/kubeedge/cloud/pkg/cloudhub/dispatcher"
	"github.com/kubeedge/kubeedge/cloud/pkg/cloudhub/session"
	"github.com/kubeedge/kubeedge/cloud/pkg/edgecontroller/controller"
	"github.com/kubeedge/kubeedge/pkg/viaduct/pkg/conn"
	"github.com/kubeedge/kubeedge/pkg/viaduct/pkg/mux"
)

type Handler interface {
	// HandleConnection is invoked when a new connection arrives
	HandleConnection(connection conn.Connection)

	// HandleMessage is invoked when a new message arrives.
	HandleMessage(container *mux.MessageContainer, writer mux.ResponseWriter)

	// OnEdgeNodeConnect is invoked when a new connection is established
	OnEdgeNodeConnect(info *model.HubInfo, connection conn.Connection) error

	// OnEdgeNodeDisconnect is invoked when a connection is lost
	OnEdgeNodeDisconnect(info *model.HubInfo, connection conn.Connection)

	// OnReadTransportErr is invoked when the connection read message err
	OnReadTransportErr(nodeID, projectID string)
}

func NewMessageHandler(
	KeepaliveInterval int,
	manager *session.Manager,
	reliableClient reliableclient.Interface,
	dispatcher dispatcher.MessageDispatcher,
	authorizer authorization.Authorizer) Handler {
	messageHandler := &messageHandler{
		KeepaliveInterval: KeepaliveInterval,
		SessionManager:    manager,
		MessageDispatcher: dispatcher,
		reliableClient:    reliableClient,
		authorizer:        authorizer,
	}

	// init handler that process upstream message
	messageHandler.initServerEntries()

	return messageHandler
}

type messageHandler struct {
	KeepaliveInterval int

	// SessionManager
	SessionManager *session.Manager

	// MessageDispatcher
	MessageDispatcher dispatcher.MessageDispatcher

	// reliableClient
	reliableClient reliableclient.Interface

	// authorizer
	authorizer authorization.Authorizer
}

// initServerEntries register handler func
func (mh *messageHandler) initServerEntries() {
	mux.Entry(mux.NewPattern("*").Op("*"), mh.HandleMessage)
}

// HandleMessage handle all the request from node
func (mh *messageHandler) HandleMessage(container *mux.MessageContainer, _ mux.ResponseWriter) {
	nodeID := container.Header.Get("node_id")
	projectID := container.Header.Get("project_id")

	// validate message
	if container.Message == nil {
		klog.Errorf("The message is nil for node: %s", nodeID)
		return
	}

	klog.V(4).Infof("[messageHandler]get msg from node(%s): %+v", nodeID, container.Message)

	hubInfo := model.HubInfo{ProjectID: projectID, NodeID: nodeID}

	if err := mh.authorizer.AdmitMessage(*container.Message, hubInfo); err != nil {
		klog.Errorf("The message is rejected by CloudHub: node=%q, message=(%+v), error=%v", nodeID, container.Message.Router, err)
		return
	}

	// dispatch upstream message
	mh.MessageDispatcher.DispatchUpstream(container.Message, &hubInfo)
}

// HandleConnection is invoked when a new connection is established
func (mh *messageHandler) HandleConnection(connection conn.Connection) {
	nodeID := connection.ConnectionState().Headers.Get("node_id")
	projectID := connection.ConnectionState().Headers.Get("project_id")

	if err := mh.authorizer.AuthenticateConnection(connection); err != nil {
		klog.Errorf("The connection is rejected by CloudHub: node=%q, error=%v", nodeID, err)
		return
	}

	if mh.SessionManager.ReachLimit() {
		klog.Errorf("Fail to serve node %s, reach node limit", nodeID)
		return
	}

	nodeInfo := &model.HubInfo{ProjectID: projectID, NodeID: nodeID}

	if err := mh.OnEdgeNodeConnect(nodeInfo, connection); err != nil {
		klog.Errorf("publish connect event for node %s, err %v", nodeInfo.NodeID, err)
		return
	}

	// start a goroutine for serving the node connection
	go func() {
		klog.Infof("edge node %s for project %s connected", nodeInfo.NodeID, nodeInfo.ProjectID)

		// init node message pool and add to the dispatcher
		nodeMessagePool := common.InitNodeMessagePool(nodeID)
		mh.MessageDispatcher.AddNodeMessagePool(nodeID, nodeMessagePool)

		keepaliveInterval := time.Duration(mh.KeepaliveInterval) * time.Second
		// create a node session for each edge node
		nodeSession := session.NewNodeSession(nodeID, projectID, connection,
			keepaliveInterval, nodeMessagePool, mh.reliableClient)
		// add node session to the session manager
		mh.SessionManager.AddSession(nodeSession)
		go func() {
			err := retry.Do(
				func() error {
					return controller.UpdateAnnotation(context.TODO(), nodeID)
				},
				retry.Delay(1*time.Second),
				retry.Attempts(3),
				retry.DelayType(retry.FixedDelay),
			)
			if err != nil {
				klog.Error(err.Error())
			}
		}()

		// start session for each edge node and it will keep running until
		// it encounters some Transport Error from underlying connection.
		nodeSession.Start()

		klog.Infof("edge node %s for project %s disConnected", nodeInfo.NodeID, nodeInfo.ProjectID)

		// clean node message pool and session
		mh.MessageDispatcher.DeleteNodeMessagePool(nodeInfo.NodeID, nodeMessagePool)
		mh.SessionManager.DeleteSession(nodeSession)
		mh.OnEdgeNodeDisconnect(nodeInfo, connection)
	}()
}

func (mh *messageHandler) OnEdgeNodeConnect(info *model.HubInfo, connection conn.Connection) error {
	err := mh.MessageDispatcher.Publish(common.ConstructConnectMessage(info, true))
	if err != nil {
		common.NotifyEventQueueError(connection, info.NodeID)
		return err
	}

	return nil
}

func (mh *messageHandler) OnEdgeNodeDisconnect(info *model.HubInfo, _ conn.Connection) {
	err := mh.MessageDispatcher.Publish(common.ConstructConnectMessage(info, false))
	if err != nil {
		klog.Errorf("fail to publish node disconnect event for node %s, reason %s", info.NodeID, err.Error())
	}
}

func (mh *messageHandler) OnReadTransportErr(nodeID, projectID string) {
	klog.Errorf("projectID %s node %s read message err", projectID, nodeID)

	nodeSession, exist := mh.SessionManager.GetSession(nodeID)
	if !exist {
		klog.Errorf("session not found for node %s", nodeID)
		return
	}

	nodeSession.Terminating()
}
