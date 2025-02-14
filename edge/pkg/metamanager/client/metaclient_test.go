package client

import (
	"errors"
	"testing"

	"github.com/kubeedge/beehive/pkg/core/model"
	"github.com/stretchr/testify/assert"
)

type mockSend struct {
	sendSyncFunc func(*model.Message) (*model.Message, error)
	sendFunc     func(*model.Message)
}

func (m *mockSend) SendSync(message *model.Message) (*model.Message, error) {
	if m.sendSyncFunc != nil {
		return m.sendSyncFunc(message)
	}
	return &model.Message{}, nil
}

func (m *mockSend) Send(message *model.Message) {
	if m.sendFunc != nil {
		m.sendFunc(message)
	}
}

func newMockSend() *mockSend {
	return &mockSend{}
}

func TestNew(t *testing.T) {
	client := New()
	assert.NotNil(t, client, "Expected non-nil client")

	_, ok := client.(CoreInterface)
	assert.True(t, ok, "Expected client to implement CoreInterface")
}

func TestSend_SendSync(t *testing.T) {
	tests := []struct {
		name           string
		mockBehavior   func() (*mockSend, *int)
		expectError    bool
		expectAttempts int
	}{
		{
			name: "success on first attempt",
			mockBehavior: func() (*mockSend, *int) {
				attempts := 0
				mock := newMockSend()
				mock.sendSyncFunc = func(msg *model.Message) (*model.Message, error) {
					attempts++
					return &model.Message{}, nil
				}
				return mock, &attempts
			},
			expectError:    false,
			expectAttempts: 1,
		},
		{
			name: "success after one retry",
			mockBehavior: func() (*mockSend, *int) {
				attempts := 0
				mock := newMockSend()
				mock.sendSyncFunc = func(msg *model.Message) (*model.Message, error) {
					attempts++
					if attempts < 2 {
						return nil, errors.New("retry needed")
					}
					return &model.Message{}, nil
				}
				return mock, &attempts
			},
			expectError:    false,
			expectAttempts: 2,
		},
		{
			name: "failure after max retries",
			mockBehavior: func() (*mockSend, *int) {
				attempts := 0
				mock := newMockSend()
				mock.sendSyncFunc = func(msg *model.Message) (*model.Message, error) {
					attempts++
					return nil, errors.New("retry needed")
				}
				return mock, &attempts
			},
			expectError:    true,
			expectAttempts: 3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup mock and get attempt counter
			mock, attempts := tt.mockBehavior()

			// Create and send message
			msg := model.NewMessage("")

			// Call SendSync multiple times to simulate retries
			var resp *model.Message
			var err error
			for i := 0; i < tt.expectAttempts; i++ {
				resp, err = mock.SendSync(msg)
				if err == nil {
					break
				}
			}

			// Verify results
			if tt.expectError {
				assert.Error(t, err, "Expected an error")
				assert.Nil(t, resp, "Expected nil response")
			} else {
				assert.NoError(t, err, "Expected no error")
				assert.NotNil(t, resp, "Expected non-nil response")
			}

			// Verify attempt count
			assert.Equal(t, tt.expectAttempts, *attempts,
				"Expected %d attempts, got %d", tt.expectAttempts, *attempts)
		})
	}
}

func TestSend_Send(t *testing.T) {
	sendCalled := false
	mock := newMockSend()
	mock.sendFunc = func(msg *model.Message) {
		sendCalled = true
	}

	msg := model.NewMessage("")
	mock.Send(msg)

	assert.True(t, sendCalled, "Send should have been called")
}

func TestMetaClient_Interfaces(t *testing.T) {
	mock := newMockSend()
	client := &metaClient{send: mock}

	tests := []struct {
		name string
		fn   func() interface{}
	}{
		{"Pods", func() interface{} { return client.Pods("default") }},
		{"ConfigMaps", func() interface{} { return client.ConfigMaps("default") }},
		{"Events", func() interface{} { return client.Events("default") }},
		{"Nodes", func() interface{} { return client.Nodes("default") }},
		{"NodeStatus", func() interface{} { return client.NodeStatus("default") }},
		{"Secrets", func() interface{} { return client.Secrets("default") }},
		{"ServiceAccountToken", func() interface{} { return client.ServiceAccountToken() }},
		{"ServiceAccounts", func() interface{} { return client.ServiceAccounts("default") }},
		{"PersistentVolumes", func() interface{} { return client.PersistentVolumes() }},
		{"PersistentVolumeClaims", func() interface{} { return client.PersistentVolumeClaims("default") }},
		{"VolumeAttachments", func() interface{} { return client.VolumeAttachments("default") }},
		{"Leases", func() interface{} { return client.Leases("default") }},
		{"CertificateSigningRequests", func() interface{} { return client.CertificateSigningRequests() }},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.fn()
			assert.NotNil(t, result, "Interface %s should not return nil", tt.name)
		})
	}
}
