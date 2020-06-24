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

package conn

import (
	"reflect"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/gorilla/websocket"

	"github.com/kubeedge/viaduct/mocks"
	"github.com/kubeedge/viaduct/pkg/api"
)

// mockSession is mock implementation of interface Session.
var mockSession *mocks.MockSession

// muxhandler is mock implementation of Mux Handler.
var muxhandler *mocks.MockHandler

// initMocks is function to initialize mocks.
func initMocks(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	mockSession = mocks.NewMockSession(mockCtrl)
	muxhandler = mocks.NewMockHandler(mockCtrl)
}

// TestNewConnection is function to test NewConnection().
func TestNewConnection(t *testing.T) {
	initMocks(t)
	wsConn := &websocket.Conn{}
	tests := []struct {
		name string
		opts *ConnectionOptions
		want Connection
	}{
		{
			name: "TestQuic",
			opts: &ConnectionOptions{ConnType: api.ProtocolTypeQuic, Base: mockSession, Handler: muxhandler},
			want: &QuicConnection{},
		},
		{
			name: "TestWS",
			opts: &ConnectionOptions{ConnType: api.ProtocolTypeWS, Base: wsConn, Handler: muxhandler},
			want: &WSConnection{},
		},
		{
			name: "TestDefault",
			opts: &ConnectionOptions{ConnType: "default"},
			want: nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := NewConnection(tt.opts); reflect.TypeOf(got) != reflect.TypeOf(tt.want) {
				t.Errorf("NewConnection() = %v, want %v", got, tt.want)
			}
		})
	}
}
