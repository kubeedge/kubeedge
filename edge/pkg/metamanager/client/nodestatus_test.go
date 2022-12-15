package client

import (
	"errors"
	"testing"

	"github.com/kubeedge/beehive/pkg/core/model"
	edgeapi "github.com/kubeedge/kubeedge/common/types"
)

type mockSender struct {
	SendInterface
}

func (s *mockSender) SendSync(message *model.Message) (*model.Message, error) {
	return &model.Message{}, nil
}

type mockErrorSender struct {
	SendInterface
}

func (s *mockErrorSender) SendSync(message *model.Message) (*model.Message, error) {
	return nil, errors.New("raise error")
}

func Test_nodeStatus_Update(t *testing.T) {
	tests := []struct {
		name      string
		namespace string
		send      SendInterface
		rsName    string
		ns        edgeapi.NodeStatusRequest
		wantErr   bool
	}{
		{
			name:      "base",
			namespace: "default",
			send:      &mockSender{},
		},
		{
			name:      "send with error",
			namespace: "default",
			send:      &mockErrorSender{},
			wantErr:   true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &nodeStatus{
				namespace: tt.namespace,
				send:      tt.send,
			}
			if err := c.Update(tt.rsName, tt.ns); (err != nil) != tt.wantErr {
				t.Errorf("nodeStatus.Update() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
