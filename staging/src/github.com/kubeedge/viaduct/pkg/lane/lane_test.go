/*
Copyright 2019 The KubeEdge Authors.

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
