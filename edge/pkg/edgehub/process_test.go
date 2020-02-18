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

package edgehub

import (
	"errors"
	"fmt"
	"reflect"
	"testing"
	"time"

	"github.com/golang/mock/gomock"

	"github.com/kubeedge/beehive/pkg/core"
	beehiveContext "github.com/kubeedge/beehive/pkg/core/context"
	"github.com/kubeedge/beehive/pkg/core/model"
	"github.com/kubeedge/kubeedge/edge/mocks/edgehub"
	module "github.com/kubeedge/kubeedge/edge/pkg/common/modules"
	"github.com/kubeedge/kubeedge/edge/pkg/edgehub/config"
)

//TestAddKeepChannel() tests the addition of channel to the syncKeeper
func TestAddKeepChannel(t *testing.T) {
	beehiveContext.InitContext(beehiveContext.MsgCtxTypeChannel)
	tests := []struct {
		name  string
		hub   *EdgeHub
		msgID string
	}{
		{
			name: "Adding a valid keep channel",
			hub: &EdgeHub{
				syncKeeper: make(map[string]chan model.Message),
			},
			msgID: "test",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.hub.addKeepChannel(tt.msgID)
			if !reflect.DeepEqual(tt.hub.syncKeeper[tt.msgID], got) {
				t.Errorf("TestController_addKeepChannel() = %v, want %v", got, tt.hub.syncKeeper[tt.msgID])
			}
		})
	}
}

//TestDeleteKeepChannel() tests the deletion of channel in the syncKeeper
func TestDeleteKeepChannel(t *testing.T) {
	beehiveContext.InitContext(beehiveContext.MsgCtxTypeChannel)
	tests := []struct {
		name  string
		hub   *EdgeHub
		msgID string
	}{
		{
			name: "Deleting a valid keep channel",
			hub: &EdgeHub{
				syncKeeper: make(map[string]chan model.Message),
			},
			msgID: "test",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.hub.addKeepChannel(tt.msgID)
			tt.hub.deleteKeepChannel(tt.msgID)
			if _, exist := tt.hub.syncKeeper[tt.msgID]; exist {
				t.Errorf("TestController_deleteKeepChannel = %v, want %v", tt.hub.syncKeeper[tt.msgID], nil)
			}
		})
	}
}

//TestIsSyncResponse() tests whether there exists a channel with the given message_id in the syncKeeper
func TestIsSyncResponse(t *testing.T) {
	beehiveContext.InitContext(beehiveContext.MsgCtxTypeChannel)
	tests := []struct {
		name  string
		hub   *EdgeHub
		msgID string
		want  bool
	}{
		{
			name: "Sync message response case",
			hub: &EdgeHub{
				syncKeeper: make(map[string]chan model.Message),
			},
			msgID: "test",
			want:  true,
		},
		{
			name: "Non sync message response  case",
			hub: &EdgeHub{
				syncKeeper: make(map[string]chan model.Message),
			},
			msgID: "",
			want:  false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.want {
				tt.hub.addKeepChannel(tt.msgID)
			}
			if got := tt.hub.isSyncResponse(tt.msgID); got != tt.want {
				t.Errorf("TestController_isSyncResponse() = %v, want %v", got, tt.want)
			}
		})
	}
}

//TestSendToKeepChannel() tests the reception of response in the syncKeep channel
func TestSendToKeepChannel(t *testing.T) {
	beehiveContext.InitContext(beehiveContext.MsgCtxTypeChannel)
	message := model.NewMessage("test_id")
	tests := []struct {
		name                string
		hub                 *EdgeHub
		message             *model.Message
		keepChannelParentID string
		expectedError       error
	}{
		{
			name: "SyncKeeper Error Case in send to keep channel",
			hub: &EdgeHub{
				syncKeeper: make(map[string]chan model.Message),
			},
			message:             message,
			keepChannelParentID: "wrong_id",
			expectedError:       fmt.Errorf("failed to get sync keeper channel, messageID:%+v", *message),
		},
		{
			name: "Negative Test Case without syncKeeper Error ",
			hub: &EdgeHub{
				syncKeeper: make(map[string]chan model.Message),
			},
			message:             model.NewMessage("test_id"),
			keepChannelParentID: "test_id",
			expectedError:       fmt.Errorf("failed to send message to sync keep channel"),
		},
		{
			name: "Send to keep channel with valid input",
			hub: &EdgeHub{
				syncKeeper: make(map[string]chan model.Message),
			},
			message:             model.NewMessage("test_id"),
			keepChannelParentID: "test_id",
			expectedError:       nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			keep := tt.hub.addKeepChannel(tt.keepChannelParentID)
			if tt.expectedError == nil {
				receive := func() {
					<-keep
				}
				go receive()
			}
			time.Sleep(1 * time.Second)
			err := tt.hub.sendToKeepChannel(*tt.message)
			if !reflect.DeepEqual(err, tt.expectedError) {
				t.Errorf("TestController_sendToKeepChannel() error = %v, expectedError %v", err, tt.expectedError)
			}
		})
	}
}

//TestDispatch() tests whether the messages are properly dispatched to their respective modules
func TestDispatch(t *testing.T) {
	beehiveContext.InitContext(beehiveContext.MsgCtxTypeChannel)
	tests := []struct {
		name          string
		hub           *EdgeHub
		message       *model.Message
		expectedError error
		isResponse    bool
	}{
		{
			name: "dispatch with valid input",
			hub: &EdgeHub{
				syncKeeper: make(map[string]chan model.Message),
			},
			message:       model.NewMessage("").BuildRouter(ModuleNameEdgeHub, module.TwinGroup, "", ""),
			expectedError: nil,
			isResponse:    false,
		},
		{
			name: "Error Case in dispatch",
			hub: &EdgeHub{
				syncKeeper: make(map[string]chan model.Message),
			},
			message:       model.NewMessage("test").BuildRouter(ModuleNameEdgeHub, module.EdgedGroup, "", ""),
			expectedError: fmt.Errorf("msg_group not found"),
			isResponse:    true,
		},
		{
			name: "Response Case in dispatch",
			hub: &EdgeHub{
				syncKeeper: make(map[string]chan model.Message),
			},
			message:       model.NewMessage("test").BuildRouter(ModuleNameEdgeHub, module.TwinGroup, "", ""),
			expectedError: nil,
			isResponse:    true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.expectedError == nil && !tt.isResponse {
				receive := func() {
					keepChannel := tt.hub.addKeepChannel(tt.message.GetParentID())
					<-keepChannel
				}
				go receive()
			}
			time.Sleep(1 * time.Second)
			err := tt.hub.dispatch(*tt.message)
			if !reflect.DeepEqual(err, tt.expectedError) {
				t.Errorf("TestController_dispatch() error = %v, wantErr %v", err, tt.expectedError)
			}
		})
	}
}

//TestRouteToEdge() is used to test whether the message received from websocket is dispatched to the required modules
func TestRouteToEdge(t *testing.T) {
	beehiveContext.InitContext(beehiveContext.MsgCtxTypeChannel)
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	mockAdapter := edgehub.NewMockAdapter(mockCtrl)
	hub := newEdgeHub(true)
	hub.chClient = mockAdapter

	tests := []struct {
		name         string
		hub          *EdgeHub
		receiveTimes int
	}{
		{
			name:         "Route to edge with proper input",
			hub:          hub,
			receiveTimes: 0,
		},
		{
			name:         "Receive Error in route to edge",
			hub:          hub,
			receiveTimes: 1,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockAdapter.EXPECT().Receive().Return(*model.NewMessage("test").BuildRouter(ModuleNameEdgeHub, module.EdgedGroup, "", ""), nil).Times(tt.receiveTimes)
			mockAdapter.EXPECT().Receive().Return(*model.NewMessage("test").BuildRouter(ModuleNameEdgeHub, module.TwinGroup, "", ""), nil).Times(tt.receiveTimes)
			mockAdapter.EXPECT().Receive().Return(*model.NewMessage(""), errors.New("Connection Refused")).Times(1)
			go tt.hub.routeToEdge()
			stop := <-tt.hub.reconnectChan
			if stop != struct{}{} {
				t.Errorf("TestRouteToEdge error got: %v want: %v", stop, struct{}{})
			}
		})
	}
}

//TestSendToCloud() tests whether the send to cloud functionality works properly
func TestSendToCloud(t *testing.T) {
	beehiveContext.InitContext(beehiveContext.MsgCtxTypeChannel)
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	mockAdapter := edgehub.NewMockAdapter(mockCtrl)

	msg := model.NewMessage("").BuildHeader("test_id", "", 1)
	msg.Header.Sync = true
	tests := []struct {
		name            string
		hub             *EdgeHub
		message         model.Message
		expectedError   error
		waitError       bool
		mockError       error
		HeartbeatPeriod int32
	}{
		{
			name: "send to cloud with proper input",
			hub: &EdgeHub{
				chClient:   mockAdapter,
				syncKeeper: make(map[string]chan model.Message),
			},
			HeartbeatPeriod: 6,
			message:         *msg,
			expectedError:   nil,
			waitError:       false,
			mockError:       nil,
		},
		{
			name: "Wait Error in send to cloud",
			hub: &EdgeHub{
				chClient:   mockAdapter,
				syncKeeper: make(map[string]chan model.Message),
			},
			HeartbeatPeriod: 3,
			message:         *msg,
			expectedError:   nil,
			waitError:       true,
			mockError:       nil,
		},
		{
			name: "Send Failure in send to cloud",
			hub: &EdgeHub{
				chClient:   mockAdapter,
				syncKeeper: make(map[string]chan model.Message),
			},
			HeartbeatPeriod: 3,
			message:         model.Message{},
			expectedError:   fmt.Errorf("failed to send message, error: Connection Refused"),
			waitError:       false,
			mockError:       errors.New("Connection Refused"),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockAdapter.EXPECT().Send(gomock.Any()).Return(tt.mockError).Times(1)
			config.Config.Heartbeat = tt.HeartbeatPeriod
			if !tt.waitError && tt.expectedError == nil {
				go tt.hub.sendToCloud(tt.message)
				time.Sleep(1 * time.Second)
				tempChannel := tt.hub.syncKeeper["test_id"]
				tempChannel <- *model.NewMessage("test_id")
				time.Sleep(1 * time.Second)
				if _, exist := tt.hub.syncKeeper["test_id"]; exist {
					t.Errorf("SendToCloud() error in receiving message")
				}
				return
			}
			err := tt.hub.sendToCloud(tt.message)
			if !reflect.DeepEqual(err, tt.expectedError) {
				t.Errorf("SendToCloud() error = %v, wantErr %v", err, tt.expectedError)
			}
			time.Sleep(time.Duration(tt.HeartbeatPeriod+2) * time.Second)
			if _, exist := tt.hub.syncKeeper["test_id"]; exist {
				t.Errorf("SendToCloud() error in waiting for timeout")
			}
		})
	}
}

//TestRouteToCloud() tests the reception of the message from the beehive framework and forwarding of that message to cloud
func TestRouteToCloud(t *testing.T) {
	beehiveContext.InitContext(beehiveContext.MsgCtxTypeChannel)
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	mockAdapter := edgehub.NewMockAdapter(mockCtrl)
	hub := newEdgeHub(true)
	hub.chClient = mockAdapter

	tests := []struct {
		name string
		hub  *EdgeHub
	}{
		{
			name: "Route to cloud with valid input",
			hub:  hub,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockAdapter.EXPECT().Send(gomock.Any()).Return(errors.New("Connection Refused")).AnyTimes()
			go tt.hub.routeToCloud()
			time.Sleep(2 * time.Second)
			core.Register(&EdgeHub{})
			beehiveContext.AddModule(ModuleNameEdgeHub)
			msg := model.NewMessage("").BuildHeader("test_id", "", 1)
			beehiveContext.Send(ModuleNameEdgeHub, *msg)
			stopChan := <-tt.hub.reconnectChan
			if stopChan != struct{}{} {
				t.Errorf("Error in route to cloud")
			}
		})
	}
}

//TestKeepalive() tests whether ping message sent to the cloud at regular intervals happens properly
func TestKeepalive(t *testing.T) {
	CertFile := "/tmp/kubeedge/certs/edge.crt"
	KeyFile := "/tmp/kubeedge/certs/edge.key"
	beehiveContext.InitContext(beehiveContext.MsgCtxTypeChannel)
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	mockAdapter := edgehub.NewMockAdapter(mockCtrl)
	tests := []struct {
		name string
		hub  *EdgeHub
	}{
		{
			name: "Heartbeat failure Case",
			hub: &EdgeHub{
				chClient:      mockAdapter,
				reconnectChan: make(chan struct{}),
			},
		},
	}
	edgeHubConfig := config.Config
	edgeHubConfig.TLSCertFile = CertFile
	edgeHubConfig.TLSPrivateKeyFile = KeyFile

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockAdapter.EXPECT().Send(gomock.Any()).Return(nil).Times(1)
			mockAdapter.EXPECT().Send(gomock.Any()).Return(errors.New("Connection Refused")).Times(1)
			go tt.hub.keepalive()
			got := <-tt.hub.reconnectChan
			if got != struct{}{} {
				t.Errorf("TestKeepalive() StopChan = %v, want %v", got, struct{}{})
			}
		})
	}
}
