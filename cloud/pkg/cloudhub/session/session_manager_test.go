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
	"testing"

	"github.com/golang/mock/gomock"

	"github.com/kubeedge/kubeedge/cloud/pkg/cloudhub/common"
	tf "github.com/kubeedge/kubeedge/cloud/pkg/cloudhub/common/testing"
	"github.com/kubeedge/kubeedge/pkg/client/clientset/versioned/fake"
	mockcon "github.com/kubeedge/viaduct/pkg/conn/testing"
)

func TestGetAddSession(t *testing.T) {
	client := &fake.Clientset{}
	nmp := common.InitNodeMessagePool(tf.TestNodeID)
	mockController := gomock.NewController(t)
	mockConn := mockcon.NewMockConnection(mockController)
	session := NewNodeSession(tf.TestNodeID, tf.TestProjectID, mockConn, tf.KeepaliveInterval, nmp, client)

	manager := NewSessionManager(10)
	manager.AddSession(session)

	actualSession, exist := manager.GetSession(tf.TestNodeID)
	if !exist {
		t.Errorf("expected session exist but got not found")
	}
	if actualSession != session {
		t.Errorf("expected: %#v, got: %#v", session, actualSession)
	}
}

func TestDeleteSession(t *testing.T) {
	client := &fake.Clientset{}
	nmp := common.InitNodeMessagePool(tf.TestNodeID)
	mockController := gomock.NewController(t)
	mockConn := mockcon.NewMockConnection(mockController)
	session := NewNodeSession(tf.TestNodeID, tf.TestProjectID, mockConn, tf.KeepaliveInterval, nmp, client)

	manager := NewSessionManager(10)
	manager.AddSession(session)
	manager.DeleteSession(session)

	_, exist := manager.GetSession(tf.TestNodeID)
	if exist {
		t.Errorf("expected session not exist but got it")
	}
}

func TestManager_KeepAliveMessage(t *testing.T) {
	client := &fake.Clientset{}
	nmp := common.InitNodeMessagePool(tf.TestNodeID)
	mockController := gomock.NewController(t)
	mockConn := mockcon.NewMockConnection(mockController)
	session := NewNodeSession(tf.TestNodeID, tf.TestProjectID, mockConn, tf.KeepaliveInterval, nmp, client)

	manager := NewSessionManager(10)
	manager.AddSession(session)

	err := manager.KeepAliveMessage(tf.TestNodeID)
	if err != nil {
		t.Errorf("expected no err but got err %v", err)
	}

	err = manager.KeepAliveMessage("no-found-node")
	if err == nil {
		t.Errorf("expected err but got nil")
	}
}

func TestManager_ReceiveMessageAck(t *testing.T) {
	client := &fake.Clientset{}
	nmp := common.InitNodeMessagePool(tf.TestNodeID)
	mockController := gomock.NewController(t)
	mockConn := mockcon.NewMockConnection(mockController)
	session := NewNodeSession(tf.TestNodeID, tf.TestProjectID, mockConn, tf.KeepaliveInterval, nmp, client)

	manager := NewSessionManager(10)
	manager.AddSession(session)

	err := manager.ReceiveMessageAck(tf.TestNodeID, "a3a93951-a717-4951-a7ad-f0b6b50d14ac")
	if err != nil {
		t.Errorf("expected no err but got err %v", err)
	}

	err = manager.ReceiveMessageAck("no-found-node", "a3a93951-a717-4951-a7ad-f0b6b50d14ac")
	if err == nil {
		t.Errorf("expected err but got nil")
	}
}
