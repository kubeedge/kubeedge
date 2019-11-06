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
	"github.com/kubeedge/beehive/pkg/core/context"
	"github.com/kubeedge/beehive/pkg/core/model"
	"github.com/kubeedge/kubeedge/edge/mocks/edgehub"
	module "github.com/kubeedge/kubeedge/edge/pkg/common/modules"
	_ "github.com/kubeedge/kubeedge/edge/pkg/devicetwin"
	"github.com/kubeedge/kubeedge/edge/pkg/edgehub/config"
)

//TestNewEdgeHubController() tests whether the EdgeHubController returned is correct or not
func TestNewEdgeHubController(t *testing.T) {
	tests := []struct {
		name             string
		want             *Controller
		controllerConfig config.ControllerConfig
	}{
		{"Testing if EdgeHubController is returned with correct values",
			&Controller{
				config: &config.ControllerConfig{
					Protocol:        "websocket",
					HeartbeatPeriod: 150 * time.Second,
					ProjectID:       "project_id",
					NodeID:          "node_id",
				},
				stopChan:   make(chan struct{}),
				syncKeeper: make(map[string]chan model.Message),
			},
			config.ControllerConfig{
				Protocol:        "websocket",
				HeartbeatPeriod: 150 * time.Second,
				ProjectID:       "project_id",
				NodeID:          "node_id",
			},
		}}
	for _, tt := range tests {
		edgeHubConfig := config.GetConfig()
		edgeHubConfig.CtrConfig = tt.controllerConfig
		t.Run(tt.name, func(t *testing.T) {
			got := NewEdgeHubController()
			if !reflect.DeepEqual(got.context, tt.want.context) {
				t.Errorf("NewEdgeHubController() Context= %v, want %v", got.context, tt.want.context)
			}
			if !reflect.DeepEqual(got.config, tt.want.config) {
				t.Errorf("NewEdgeHubController() Config = %v, want %v", got.config, tt.want.config)
			}
			if !reflect.DeepEqual(got.chClient, tt.want.chClient) {
				t.Errorf("NewEdgeHubController() chClient = %v, want %v", got.chClient, tt.want.chClient)
			}
			if !reflect.DeepEqual(got.syncKeeper, tt.want.syncKeeper) {
				t.Errorf("NewEdgeHubController() SyncKeeper = %v, want %v", got.syncKeeper, tt.want.syncKeeper)
			}
		})
	}
}

//TestAddKeepChannel() tests the addition of channel to the syncKeeper
func TestAddKeepChannel(t *testing.T) {
	tests := []struct {
		name       string
		controller *Controller
		msgID      string
	}{
		{
			name: "Adding a valid keep channel",
			controller: &Controller{
				syncKeeper: make(map[string]chan model.Message),
			},
			msgID: "test",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.controller.addKeepChannel(tt.msgID)
			if !reflect.DeepEqual(tt.controller.syncKeeper[tt.msgID], got) {
				t.Errorf("TestController_addKeepChannel() = %v, want %v", got, tt.controller.syncKeeper[tt.msgID])
			}
		})
	}
}

//TestDeleteKeepChannel() tests the deletion of channel in the syncKeeper
func TestDeleteKeepChannel(t *testing.T) {
	tests := []struct {
		name       string
		controller *Controller
		msgID      string
	}{
		{
			name: "Deleting a valid keep channel",
			controller: &Controller{
				syncKeeper: make(map[string]chan model.Message),
			},
			msgID: "test",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.controller.addKeepChannel(tt.msgID)
			tt.controller.deleteKeepChannel(tt.msgID)
			if _, exist := tt.controller.syncKeeper[tt.msgID]; exist {
				t.Errorf("TestController_deleteKeepChannel = %v, want %v", tt.controller.syncKeeper[tt.msgID], nil)
			}
		})
	}
}

//TestIsSyncResponse() tests whether there exists a channel with the given message_id in the syncKeeper
func TestIsSyncResponse(t *testing.T) {
	tests := []struct {
		name       string
		controller *Controller
		msgID      string
		want       bool
	}{
		{
			name: "Sync message response case",
			controller: &Controller{
				syncKeeper: make(map[string]chan model.Message),
			},
			msgID: "test",
			want:  true,
		},
		{
			name: "Non sync message response  case",
			controller: &Controller{
				syncKeeper: make(map[string]chan model.Message),
			},
			msgID: "",
			want:  false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.want {
				tt.controller.addKeepChannel(tt.msgID)
			}
			if got := tt.controller.isSyncResponse(tt.msgID); got != tt.want {
				t.Errorf("TestController_isSyncResponse() = %v, want %v", got, tt.want)
			}
		})
	}
}

//TestSendToKeepChannel() tests the reception of response in the syncKeep channel
func TestSendToKeepChannel(t *testing.T) {
	message := model.NewMessage("test_id")
	tests := []struct {
		name                string
		controller          *Controller
		message             *model.Message
		keepChannelParentID string
		expectedError       error
	}{
		{
			name: "SyncKeeper Error Case in send to keep channel",
			controller: &Controller{
				context:    context.GetContext(context.MsgCtxTypeChannel),
				syncKeeper: make(map[string]chan model.Message),
			},
			message:             message,
			keepChannelParentID: "wrong_id",
			expectedError:       fmt.Errorf("failed to get sync keeper channel, messageID:%+v", *message),
		},
		{
			name: "Negative Test Case without syncKeeper Error ",
			controller: &Controller{
				context:    context.GetContext(context.MsgCtxTypeChannel),
				syncKeeper: make(map[string]chan model.Message),
			},
			message:             model.NewMessage("test_id"),
			keepChannelParentID: "test_id",
			expectedError:       fmt.Errorf("failed to send message to sync keep channel"),
		},
		{
			name: "Send to keep channel with valid input",
			controller: &Controller{
				context:    context.GetContext(context.MsgCtxTypeChannel),
				syncKeeper: make(map[string]chan model.Message),
			},
			message:             model.NewMessage("test_id"),
			keepChannelParentID: "test_id",
			expectedError:       nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			keep := tt.controller.addKeepChannel(tt.keepChannelParentID)
			if tt.expectedError == nil {
				receive := func() {
					<-keep
				}
				go receive()
			}
			time.Sleep(1 * time.Second)
			err := tt.controller.sendToKeepChannel(*tt.message)
			if !reflect.DeepEqual(err, tt.expectedError) {
				t.Errorf("TestController_sendToKeepChannel() error = %v, expectedError %v", err, tt.expectedError)
			}
		})
	}
}

//TestDispatch() tests whether the messages are properly dispatched to their respective modules
func TestDispatch(t *testing.T) {
	tests := []struct {
		name          string
		controller    *Controller
		message       *model.Message
		expectedError error
		isResponse    bool
	}{
		{
			name: "dispatch with valid input",
			controller: &Controller{
				context:    context.GetContext(context.MsgCtxTypeChannel),
				syncKeeper: make(map[string]chan model.Message),
			},
			message:       model.NewMessage("").BuildRouter(ModuleNameEdgeHub, module.TwinGroup, "", ""),
			expectedError: nil,
			isResponse:    false,
		},
		{
			name: "Error Case in dispatch",
			controller: &Controller{
				context:    context.GetContext(context.MsgCtxTypeChannel),
				syncKeeper: make(map[string]chan model.Message),
			},
			message:       model.NewMessage("test").BuildRouter(ModuleNameEdgeHub, module.EdgedGroup, "", ""),
			expectedError: fmt.Errorf("msg_group not found"),
			isResponse:    true,
		},
		{
			name: "Response Case in dispatch",
			controller: &Controller{
				context:    context.GetContext(context.MsgCtxTypeChannel),
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
					keepChannel := tt.controller.addKeepChannel(tt.message.GetParentID())
					<-keepChannel
				}
				go receive()
			}
			time.Sleep(1 * time.Second)
			err := tt.controller.dispatch(*tt.message)
			if !reflect.DeepEqual(err, tt.expectedError) {
				t.Errorf("TestController_dispatch() error = %v, wantErr %v", err, tt.expectedError)
			}
		})
	}
}

//TestRouteToEdge() is used to test whether the message received from websocket is dispatched to the required modules
func TestRouteToEdge(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	mockAdapter := edgehub.NewMockAdapter(mockCtrl)
	tests := []struct {
		name         string
		controller   *Controller
		receiveTimes int
	}{
		{
			name: "Route to edge with proper input",
			controller: &Controller{
				context:    context.GetContext(context.MsgCtxTypeChannel),
				chClient:   mockAdapter,
				syncKeeper: make(map[string]chan model.Message),
				stopChan:   make(chan struct{}),
			},
			receiveTimes: 0,
		},
		{
			name: "Receive Error in route to edge",
			controller: &Controller{
				context:    context.GetContext(context.MsgCtxTypeChannel),
				chClient:   mockAdapter,
				syncKeeper: make(map[string]chan model.Message),
				stopChan:   make(chan struct{}),
			},
			receiveTimes: 1,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockAdapter.EXPECT().Receive().Return(*model.NewMessage("test").BuildRouter(ModuleNameEdgeHub, module.EdgedGroup, "", ""), nil).Times(tt.receiveTimes)
			mockAdapter.EXPECT().Receive().Return(*model.NewMessage("test").BuildRouter(ModuleNameEdgeHub, module.TwinGroup, "", ""), nil).Times(tt.receiveTimes)
			mockAdapter.EXPECT().Receive().Return(*model.NewMessage(""), errors.New("Connection Refused")).Times(1)
			go tt.controller.routeToEdge()
			stop := <-tt.controller.stopChan
			if stop != struct{}{} {
				t.Errorf("TestRouteToEdge error got: %v want: %v", stop, struct{}{})
			}
		})
	}
}

//TestSendToCloud() tests whether the send to cloud functionality works properly
func TestSendToCloud(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	mockAdapter := edgehub.NewMockAdapter(mockCtrl)

	msg := model.NewMessage("").BuildHeader("test_id", "", 1)
	msg.Header.Sync = true
	tests := []struct {
		name          string
		controller    *Controller
		message       model.Message
		expectedError error
		waitError     bool
		mockError     error
	}{
		{
			name: "send to cloud with proper input",
			controller: &Controller{
				context:  context.GetContext(context.MsgCtxTypeChannel),
				chClient: mockAdapter,
				config: &config.ControllerConfig{
					Protocol:        "websocket",
					HeartbeatPeriod: 6 * time.Second,
				},
				syncKeeper: make(map[string]chan model.Message),
			},
			message:       *msg,
			expectedError: nil,
			waitError:     false,
			mockError:     nil,
		},
		{
			name: "Wait Error in send to cloud",
			controller: &Controller{
				chClient: mockAdapter,
				config: &config.ControllerConfig{
					Protocol:        "websocket",
					HeartbeatPeriod: 3 * time.Second,
				},
				syncKeeper: make(map[string]chan model.Message),
			},
			message:       *msg,
			expectedError: nil,
			waitError:     true,
			mockError:     nil,
		},
		{
			name: "Send Failure in send to cloud",
			controller: &Controller{
				chClient: mockAdapter,
				config: &config.ControllerConfig{
					HeartbeatPeriod: 3 * time.Second,
				},
				syncKeeper: make(map[string]chan model.Message),
			},
			message:       model.Message{},
			expectedError: fmt.Errorf("failed to send message, error: Connection Refused"),
			waitError:     false,
			mockError:     errors.New("Connection Refused"),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockAdapter.EXPECT().Send(gomock.Any()).Return(tt.mockError).Times(1)
			if !tt.waitError && tt.expectedError == nil {
				go tt.controller.sendToCloud(tt.message)
				time.Sleep(1 * time.Second)
				tempChannel := tt.controller.syncKeeper["test_id"]
				tempChannel <- *model.NewMessage("test_id")
				time.Sleep(1 * time.Second)
				if _, exist := tt.controller.syncKeeper["test_id"]; exist {
					t.Errorf("SendToCloud() error in receiving message")
				}
				return
			}
			err := tt.controller.sendToCloud(tt.message)
			if !reflect.DeepEqual(err, tt.expectedError) {
				t.Errorf("SendToCloud() error = %v, wantErr %v", err, tt.expectedError)
			}
			time.Sleep(tt.controller.config.HeartbeatPeriod + 2*time.Second)
			if _, exist := tt.controller.syncKeeper["test_id"]; exist {
				t.Errorf("SendToCloud() error in waiting for timeout")
			}
		})
	}
}

//TestRouteToCloud() tests the reception of the message from the beehive framework and forwarding of that message to cloud
func TestRouteToCloud(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	mockAdapter := edgehub.NewMockAdapter(mockCtrl)
	testContext := context.GetContext(context.MsgCtxTypeChannel)
	tests := []struct {
		name       string
		controller *Controller
	}{
		{
			name: "Route to cloud with valid input",
			controller: &Controller{
				context:  testContext,
				chClient: mockAdapter,
				stopChan: make(chan struct{}),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockAdapter.EXPECT().Send(gomock.Any()).Return(errors.New("Connection Refused")).AnyTimes()
			go tt.controller.routeToCloud()
			time.Sleep(2 * time.Second)
			core.Register(&EdgeHub{})
			testContext.AddModule(ModuleNameEdgeHub)
			msg := model.NewMessage("").BuildHeader("test_id", "", 1)
			testContext.Send(ModuleNameEdgeHub, *msg)
			stopChan := <-tt.controller.stopChan
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
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	mockAdapter := edgehub.NewMockAdapter(mockCtrl)
	tests := []struct {
		name       string
		controller *Controller
	}{
		{
			name: "Heartbeat failure Case",
			controller: &Controller{
				config: &config.ControllerConfig{
					Protocol:  "websocket",
					ProjectID: "foo",
					NodeID:    "bar",
				},
				chClient: mockAdapter,
				stopChan: make(chan struct{}),
			},
		},
	}
	edgeHubConfig := config.GetConfig()
	edgeHubConfig.WSConfig = config.WebSocketConfig{
		CertFilePath: CertFile,
		KeyFilePath:  KeyFile,
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockAdapter.EXPECT().Send(gomock.Any()).Return(nil).Times(1)
			mockAdapter.EXPECT().Send(gomock.Any()).Return(errors.New("Connection Refused")).Times(1)
			go tt.controller.keepalive()
			got := <-tt.controller.stopChan
			if got != struct{}{} {
				t.Errorf("TestKeepalive() StopChan = %v, want %v", got, struct{}{})
			}
		})
	}
}
