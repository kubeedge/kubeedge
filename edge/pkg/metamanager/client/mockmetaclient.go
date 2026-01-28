package client

import (
	"fmt"

	"github.com/kubeedge/beehive/pkg/core/model"
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
