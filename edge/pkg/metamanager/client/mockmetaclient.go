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

package client

import (
	"fmt"

	"github.com/kubeedge/beehive/pkg/core/model"
	"github.com/kubeedge/kubeedge/edge/pkg/metamanager/dao/models"
)

const testNamespace = "default"

// mockSendInterface is a mock implementation of SendInterface
type mockSendInterface struct {
	sendSyncFunc func(*model.Message) (*model.Message, error)
	sendFunc     func(*model.Message)
}

func (m *mockSendInterface) SendSync(msg *model.Message) (*model.Message, error) {
	if m.sendSyncFunc != nil {
		return m.sendSyncFunc(msg)
	}
	return nil, fmt.Errorf("not implemented")
}

func (m *mockSendInterface) Send(msg *model.Message) {
	if m.sendFunc != nil {
		m.sendFunc(msg)
	}
}

func newMockSend() SendInterface {
	return &mockSendInterface{}
}

// NewSecretsWithMetaService creates a new secrets instance with custom meta service (for testing)
func NewSecretsWithMetaService(namespace string, s SendInterface, metaService MetaServiceInterface) *secrets {
	return &secrets{
		send:        s,
		namespace:   namespace,
		metaService: metaService,
	}
}

// NewConfigMapsWithMetaService creates a new ConfigMap instance with custom meta service (for testing)
func NewConfigMapsWithMetaService(namespace string, s SendInterface, metaService MetaServiceInterface) *configMaps {
	return &configMaps{
		send:        s,
		namespace:   namespace,
		metaService: metaService,
	}
}

// MockMetaService is a mock implementation of MetaServiceInterface
type MockMetaService struct {
	QueryMetaFunc      func(key, value string) (*[]string, error)
	InsertOrUpdateFunc func(meta *models.Meta) error
}

func (m *MockMetaService) QueryMeta(key, value string) (*[]string, error) {
	if m.QueryMetaFunc != nil {
		return m.QueryMetaFunc(key, value)
	}
	return &[]string{}, nil
}

func (m *MockMetaService) InsertOrUpdate(meta *models.Meta) error {
	if m.InsertOrUpdateFunc != nil {
		return m.InsertOrUpdateFunc(meta)
	}
	return nil
}
