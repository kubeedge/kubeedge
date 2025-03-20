/*
Copyright 2025 The KubeEdge Authors.

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
	"testing"

	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/rand"

	beehivemodel "github.com/kubeedge/beehive/pkg/core/model"
	edgecon "github.com/kubeedge/kubeedge/cloud/pkg/edgecontroller/constants"
)

type TestMessageObj struct {
	metav1.TypeMeta
	metav1.ObjectMeta
}

func (in *TestMessageObj) DeepCopyObject() runtime.Object {
	return &TestMessageObj{
		TypeMeta:   in.TypeMeta,
		ObjectMeta: in.ObjectMeta,
	}
}

func TestInitNodeMessagePool(t *testing.T) {
	nodeID := "test-node-" + rand.String(5)
	pool := InitNodeMessagePool(nodeID)

	assert.NotNil(t, pool.AckMessageStore, "AckMessageStore should not be nil")
	assert.NotNil(t, pool.AckMessageQueue, "AckMessageQueue should not be nil")
	assert.NotNil(t, pool.NoAckMessageStore, "NoAckMessageStore should not be nil")
	assert.NotNil(t, pool.NoAckMessageQueue, "NoAckMessageQueue should not be nil")

	assert.False(t, pool.AckMessageQueue.ShuttingDown(), "AckMessageQueue should not be shutting down when initialized")
	assert.False(t, pool.NoAckMessageQueue.ShuttingDown(), "NoAckMessageQueue should not be shutting down when initialized")
}

func TestGetAckMessage(t *testing.T) {
	nodeID := "test-node-" + rand.String(5)
	pool := InitNodeMessagePool(nodeID)

	nonExistentKey := "non-existent-key"
	msg, err := pool.GetAckMessage(nonExistentKey)
	assert.Error(t, err, "Should return error for non-existent key")
	assert.Nil(t, msg, "Message should be nil when not found")
	assert.Contains(t, err.Error(), "not found", "Error should indicate message not found")

	testObj := &TestMessageObj{
		ObjectMeta: metav1.ObjectMeta{
			UID: "test-uid",
		},
	}
	testMsg := beehivemodel.NewMessage("")
	testMsg.BuildRouter("source", edgecon.GroupResource, "resource", "operation")
	testMsg.Content = testObj

	key, err := AckMessageKeyFunc(testMsg)
	assert.NoError(t, err, "AckMessageKeyFunc should not fail")

	err = pool.AckMessageStore.Add(testMsg)
	assert.NoError(t, err, "Adding message to store should not fail")

	retrievedMsg, err := pool.GetAckMessage(key)
	assert.NoError(t, err, "Getting existing message should not fail")
	assert.NotNil(t, retrievedMsg, "Retrieved message should not be nil")
	assert.Equal(t, testMsg.GetID(), retrievedMsg.GetID(), "Retrieved message ID should match original")

	mockErrStore := &mockStore{
		getByKeyFunc: func(key string) (interface{}, bool, error) {
			return nil, false, fmt.Errorf("mock error")
		},
	}
	origAckStore := pool.AckMessageStore
	pool.AckMessageStore = mockErrStore
	_, err = pool.GetAckMessage("any-key")
	assert.Error(t, err, "Should return error when store.GetByKey fails")
	assert.Contains(t, err.Error(), "err:", "Error should include store error message")
	pool.AckMessageStore = origAckStore

	mockInvalidTypeStore := &mockStore{
		getByKeyFunc: func(key string) (interface{}, bool, error) {
			return "not-a-message", true, nil
		},
	}
	pool.AckMessageStore = mockInvalidTypeStore
	_, err = pool.GetAckMessage("invalid-type-key")
	assert.Error(t, err, "Should return error for invalid message type")
	assert.Contains(t, err.Error(), "invalid", "Error should indicate invalid type")
	pool.AckMessageStore = origAckStore

	mockNilMsgStore := &mockStore{
		getByKeyFunc: func(key string) (interface{}, bool, error) {
			return nil, true, nil
		},
	}
	pool.AckMessageStore = mockNilMsgStore
	_, err = pool.GetAckMessage("nil-key")
	assert.Error(t, err, "Should return error for nil message")
	assert.Contains(t, err.Error(), "nil", "Error should indicate message is nil")
	pool.AckMessageStore = origAckStore
}

func TestGetNoAckMessage(t *testing.T) {
	nodeID := "test-node-" + rand.String(5)
	pool := InitNodeMessagePool(nodeID)

	nonExistentKey := "non-existent-key"
	msg, err := pool.GetNoAckMessage(nonExistentKey)
	assert.Error(t, err, "Should return error for non-existent key")
	assert.Nil(t, msg, "Message should be nil when not found")
	assert.Contains(t, err.Error(), "not found", "Error should indicate message not found")

	testMsg := beehivemodel.NewMessage("")
	testMsg.BuildRouter("source", "group", "resource", "operation")

	key, err := NoAckMessageKeyFunc(testMsg)
	assert.NoError(t, err, "NoAckMessageKeyFunc should not fail")
	assert.Equal(t, testMsg.GetID(), key, "NoAckMessageKeyFunc should return message ID")

	err = pool.NoAckMessageStore.Add(testMsg)
	assert.NoError(t, err, "Adding message to store should not fail")

	retrievedMsg, err := pool.GetNoAckMessage(key)
	assert.NoError(t, err, "Getting existing message should not fail")
	assert.NotNil(t, retrievedMsg, "Retrieved message should not be nil")
	assert.Equal(t, testMsg.GetID(), retrievedMsg.GetID(), "Retrieved message ID should match original")

	mockErrStore := &mockStore{
		getByKeyFunc: func(key string) (interface{}, bool, error) {
			return nil, false, fmt.Errorf("mock error")
		},
	}
	origNoAckStore := pool.NoAckMessageStore
	pool.NoAckMessageStore = mockErrStore
	_, err = pool.GetNoAckMessage("any-key")
	assert.Error(t, err, "Should return error when store.GetByKey fails")
	assert.Contains(t, err.Error(), "err:", "Error should include store error message")
	pool.NoAckMessageStore = origNoAckStore

	mockInvalidTypeStore := &mockStore{
		getByKeyFunc: func(key string) (interface{}, bool, error) {
			return "not-a-message", true, nil
		},
	}
	pool.NoAckMessageStore = mockInvalidTypeStore
	_, err = pool.GetNoAckMessage("invalid-type-key")
	assert.Error(t, err, "Should return error for invalid message type")
	assert.Contains(t, err.Error(), "invalid", "Error should indicate invalid type")
	pool.NoAckMessageStore = origNoAckStore

	mockNilMsgStore := &mockStore{
		getByKeyFunc: func(key string) (interface{}, bool, error) {
			return nil, true, nil
		},
	}
	pool.NoAckMessageStore = mockNilMsgStore
	_, err = pool.GetNoAckMessage("nil-key")
	assert.Error(t, err, "Should return error for nil message")
	assert.Contains(t, err.Error(), "nil", "Error should indicate message is nil")
	pool.NoAckMessageStore = origNoAckStore
}

func TestShutDown(t *testing.T) {
	nodeID := "test-node-" + rand.String(5)
	pool := InitNodeMessagePool(nodeID)

	assert.NotPanics(t, func() {
		pool.ShutDown()
	}, "ShutDown should not panic")

	assert.True(t, pool.AckMessageQueue.ShuttingDown(), "AckMessageQueue should be shutting down after ShutDown")
	assert.True(t, pool.NoAckMessageQueue.ShuttingDown(), "NoAckMessageQueue should be shutting down after ShutDown")

	assert.NotPanics(t, func() {
		pool.ShutDown()
	}, "Calling ShutDown multiple times should not panic")
}

type mockStore struct {
	getByKeyFunc func(key string) (interface{}, bool, error)
}

func (m *mockStore) Add(_ interface{}) error {
	return nil
}

func (m *mockStore) Update(_ interface{}) error {
	return nil
}

func (m *mockStore) Delete(_ interface{}) error {
	return nil
}

func (m *mockStore) List() []interface{} {
	return nil
}

func (m *mockStore) ListKeys() []string {
	return nil
}

func (m *mockStore) Get(_ interface{}) (interface{}, bool, error) {
	return nil, false, nil
}

func (m *mockStore) GetByKey(key string) (interface{}, bool, error) {
	if m.getByKeyFunc != nil {
		return m.getByKeyFunc(key)
	}
	return nil, false, nil
}

func (m *mockStore) Replace([]interface{}, string) error {
	return nil
}

func (m *mockStore) Resync() error {
	return nil
}
