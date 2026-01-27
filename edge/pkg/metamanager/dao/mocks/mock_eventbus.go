/*
Copyright 2026 The KubeEdge Authors.

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

package mocks

import (
	"sync"
)

// MockEventBusService provides a mock implementation of EventBusService for testing
type MockEventBusService struct {
	mu sync.RWMutex

	// InsertTopicsFunc can be overridden for testing
	InsertTopicsFunc func(topic string) error

	// DeleteTopicsByKeyFunc can be overridden for testing
	DeleteTopicsByKeyFunc func(key string) error

	// QueryAllTopicsFunc can be overridden for testing
	QueryAllTopicsFunc func() (*[]string, error)
}

// NewMockEventBusService creates a new mock EventBus service with default implementations
func NewMockEventBusService() *MockEventBusService {
	return &MockEventBusService{
		InsertTopicsFunc: func(topic string) error {
			return nil
		},
		DeleteTopicsByKeyFunc: func(key string) error {
			return nil
		},
		QueryAllTopicsFunc: func() (*[]string, error) {
			return nil, nil
		},
	}
}

// InsertTopics mocks the InsertTopics method
func (m *MockEventBusService) InsertTopics(topic string) error {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.InsertTopicsFunc(topic)
}

// DeleteTopicsByKey mocks the DeleteTopicsByKey method
func (m *MockEventBusService) DeleteTopicsByKey(key string) error {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.DeleteTopicsByKeyFunc(key)
}

// QueryAllTopics mocks the QueryAllTopics method
func (m *MockEventBusService) QueryAllTopics() (*[]string, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.QueryAllTopicsFunc()
}
