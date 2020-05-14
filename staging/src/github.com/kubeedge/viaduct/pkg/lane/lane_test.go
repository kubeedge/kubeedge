package lane

import (
	"reflect"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/gorilla/websocket"

	"github.com/kubeedge/viaduct/mocks"
	"github.com/kubeedge/viaduct/pkg/api"
)

// mockStream is mock of interface Stream.
var mockStream *mocks.MockStream

// initMocks is function to initialize mocks.
func initMocks(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	mockStream = mocks.NewMockStream(mockCtrl)
}

// TestNewLane is function to test NewLane().
func TestNewLane(t *testing.T) {
	initMocks(t)
	wsConn := &websocket.Conn{}
	tests := []struct {
		name      string
		protoType string
		van       interface{}
		want      Lane
	}{

		{
			name:      "TestQuic",
			protoType: api.ProtocolTypeQuic,
			van:       mockStream,
			want:      &QuicLane{},
		},
		{
			name:      "TestWS",
			protoType: api.ProtocolTypeWS,
			van:       wsConn,
			want:      &WSLane{},
		},
		{
			name:      "TestDefault",
			protoType: "default",
			van:       "",
			want:      nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := NewLane(tt.protoType, tt.van); reflect.TypeOf(got) != reflect.TypeOf(tt.want) {
				t.Errorf("NewLane() = %v, want %v", got, tt.want)
			}
		})
	}
}
