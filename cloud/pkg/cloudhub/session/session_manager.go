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
	"fmt"
	"sync"
	"sync/atomic"

	"k8s.io/klog/v2"
)

type Manager struct {
	// NodeNumber is the number of currently connected edge
	// nodes for single cloudHub instance
	NodeNumber int32
	// NodeLimit is the maximum number of edge nodes that can
	// connected to single cloudHub instance
	NodeLimit int32
	// NodeSessions maps a node ID to NodeSession
	NodeSessions sync.Map
}

// NewSessionManager initializes a new SessionManager
func NewSessionManager(nodeLimit int32) *Manager {
	return &Manager{
		NodeLimit:    nodeLimit,
		NodeSessions: sync.Map{},
	}
}

// AddSession add node session to the session manager
func (sm *Manager) AddSession(session *NodeSession) {
	nodeID := session.nodeID

	ons, exists := sm.NodeSessions.LoadAndDelete(nodeID)
	if exists {
		if oldSession, ok := ons.(*NodeSession); ok {
			klog.Warningf("session exists for %s, close old session", nodeID)
			oldSession.Terminating()
			atomic.AddInt32(&sm.NodeNumber, -1)
		}
	}

	sm.NodeSessions.Store(nodeID, session)
	atomic.AddInt32(&sm.NodeNumber, 1)
}

// DeleteSession delete the node session from session manager
func (sm *Manager) DeleteSession(session *NodeSession) {
	cacheSession, exist := sm.GetSession(session.nodeID)
	if !exist {
		klog.Warningf("session not found for node %s", session.nodeID)
		return
	}

	// This usually happens when the node is disconnect then quickly reconnect
	if cacheSession != session {
		klog.Warningf("the session %s already deleted", session.nodeID)
		return
	}

	sm.NodeSessions.Delete(session.nodeID)
	atomic.AddInt32(&sm.NodeNumber, -1)
}

// GetSession get the node session for the node
func (sm *Manager) GetSession(nodeID string) (*NodeSession, bool) {
	ons, exists := sm.NodeSessions.Load(nodeID)
	if exists {
		return ons.(*NodeSession), true
	}

	return nil, false
}

// ReachLimit checks whether the connected nodes exceeds the node limit number
func (sm *Manager) ReachLimit() bool {
	return atomic.LoadInt32(&sm.NodeNumber) >= sm.NodeLimit
}

// KeepAliveMessage receive keepalive message from edge node
func (sm *Manager) KeepAliveMessage(nodeID string) error {
	session, exist := sm.GetSession(nodeID)
	if !exist {
		return fmt.Errorf("session not found for node %s", nodeID)
	}

	session.KeepAliveMessage()
	return nil
}

// ReceiveMessageAck receive the message ack from edge node
func (sm *Manager) ReceiveMessageAck(nodeID, parentID string) error {
	session, exist := sm.GetSession(nodeID)
	if !exist {
		return fmt.Errorf("session not found for node %s", nodeID)
	}

	session.ReceiveMessageAck(parentID)
	return nil
}
