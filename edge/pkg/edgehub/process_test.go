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
	"testing"
	"time"

	"github.com/agiledragon/gomonkey/v2"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"

	"github.com/kubeedge/beehive/pkg/common"
	"github.com/kubeedge/beehive/pkg/core"
	beehiveContext "github.com/kubeedge/beehive/pkg/core/context"
	"github.com/kubeedge/beehive/pkg/core/model"
	cloudmodules "github.com/kubeedge/kubeedge/cloud/pkg/common/modules"
	"github.com/kubeedge/kubeedge/edge/mocks/edgehub"
	"github.com/kubeedge/kubeedge/edge/pkg/common/message"
	"github.com/kubeedge/kubeedge/edge/pkg/common/modules"
	"github.com/kubeedge/kubeedge/edge/pkg/edgehub/config"
	msghandler "github.com/kubeedge/kubeedge/edge/pkg/edgehub/messagehandler"
)

func init() {
	add := &common.ModuleInfo{
		ModuleName: modules.EdgeHubModuleName,
		ModuleType: common.MsgCtxTypeChannel,
	}

	beehiveContext.InitContext([]string{common.MsgCtxTypeChannel})

	beehiveContext.AddModule(add)
	beehiveContext.AddModuleGroup(modules.EdgeHubModuleName, modules.EdgeHubModuleName)
}

// TestDispatch() tests whether the messages are properly dispatched to their respective modules
func TestDispatch(t *testing.T) {
	var handleLog string

	patches := gomonkey.NewPatches()
	defer patches.Reset()

	patches.ApplyFunc(beehiveContext.SendToGroup, func(group string, _msg model.Message) {
		handleLog = fmt.Sprintf("send to %s group", group)
	})
	patches.ApplyFunc(beehiveContext.Send, func(module string, _msg model.Message) {
		handleLog = fmt.Sprintf("send to %s module", module)
	})
	patches.ApplyFunc(beehiveContext.SendResp, func(msg model.Message) {
		handleLog = fmt.Sprintf("send response to %s", msg.GetParentID())
	})

	tests := []struct {
		name              string
		message           *model.Message
		expectedHandleLog string
		expectedError     string
	}{
		{
			name: "error case in dispatch",
			message: model.NewMessage("test").
				BuildRouter(modules.EdgeHubModuleName, modules.EdgedGroup, "", ""),
			expectedError: "failed to handle message, no handler found for the message, message group: edged",
		},
		{
			name: "dispatch to twin group and not support response",
			message: model.NewMessage("parent").
				BuildRouter(modules.EdgeHubModuleName, modules.TwinGroup, "", ""),
			expectedHandleLog: "send to twin group",
		},
		{
			name: "dispatch to event bus module",
			message: model.NewMessage("").
				BuildRouter("router_eventbus", message.UserGroupName, "", ""),
			expectedHandleLog: "send to eventbus module",
		},
		{
			name: "dispatch to service bus module",
			message: model.NewMessage("").
				BuildRouter("router_servicebus", message.UserGroupName, "", ""),
			expectedHandleLog: "send to servicebus module",
		},
		{
			name: "dispatch to service bus module to call response",
			message: model.NewMessage("parent").
				BuildRouter("router_servicebus", message.UserGroupName, "", ""),
			expectedHandleLog: "send response to parent",
		},
		{
			name: "dispatch to meta group with resource message group",
			message: model.NewMessage("").
				BuildRouter(modules.EdgeHubModuleName, message.ResourceGroupName, "", ""),
			expectedHandleLog: "send to meta group",
		},
		{
			name: "dispatch to meta group with func message group",
			message: model.NewMessage("").
				BuildRouter(modules.EdgeHubModuleName, message.FuncGroupName, "", ""),
			expectedHandleLog: "send to meta group",
		},
		{
			name: "dispatch to meta group to call response",
			message: model.NewMessage("parent").
				BuildRouter(modules.EdgeHubModuleName, message.FuncGroupName, "", ""),
			expectedHandleLog: "send response to parent",
		},
		{
			name: "dispatch to taskmassage module and not support response",
			message: model.NewMessage("parent").
				BuildRouter(modules.EdgeHubModuleName, cloudmodules.TaskManagerModuleName, "", ""),
			expectedHandleLog: "send to taskmanager module",
		},
	}

	msghandler.RegisterHandlers()
	hub := &EdgeHub{}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handleLog = ""
			err := hub.dispatch(*tt.message)
			if tt.expectedError != "" {
				require.ErrorContains(t, err, tt.expectedError)
			} else {
				require.NoError(t, err)
				require.Equal(t, tt.expectedHandleLog, handleLog)
			}

			// New logic, ignore it
			if tt.message.GetGroup() == cloudmodules.TaskManagerModuleName {
				return
			}

			// Verify that logic has not changed due to changes
			handleLog = ""
			checkSameAsHistoricalDispatch(*tt.message)
			if tt.expectedError != "" {
				require.False(t, checkSameAsHistoricalDispatchFilter(*tt.message))
			} else {
				require.True(t, checkSameAsHistoricalDispatchFilter(*tt.message))
				require.Equal(t, tt.expectedHandleLog, handleLog)
			}
		})
	}
}

func checkSameAsHistoricalDispatchFilter(msg model.Message) bool {
	group := msg.GetGroup()
	return group == message.ResourceGroupName || group == modules.TwinGroup ||
		group == message.FuncGroupName || group == message.UserGroupName
}

func checkSameAsHistoricalDispatch(msg model.Message) {
	group := msg.GetGroup()
	md := ""
	switch group {
	case message.ResourceGroupName:
		md = modules.MetaGroup
	case modules.TwinGroup:
		md = modules.TwinGroup
	case message.FuncGroupName:
		md = modules.MetaGroup
	case message.UserGroupName:
		md = modules.BusGroup
	}

	if group == modules.TwinGroup {
		beehiveContext.SendToGroup(md, msg)
		return
	}

	if msg.GetParentID() != "" {
		beehiveContext.SendResp(msg)
		return
	}
	if group == message.UserGroupName && msg.GetSource() == "router_eventbus" {
		beehiveContext.Send(modules.EventBusModuleName, msg)
	} else if group == message.UserGroupName && msg.GetSource() == "router_servicebus" {
		beehiveContext.Send(modules.ServiceBusModuleName, msg)
	} else {
		beehiveContext.SendToGroup(md, msg)
	}
}

// TestRouteToEdge() is used to test whether the message received from websocket is dispatched to the required modules
func TestRouteToEdge(t *testing.T) {
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
			mockAdapter.EXPECT().Receive().
				Return(*model.NewMessage("test").BuildRouter(modules.EdgeHubModuleName, modules.EdgedGroup, "", ""), nil).
				Times(tt.receiveTimes)
			mockAdapter.EXPECT().Receive().
				Return(*model.NewMessage("test").BuildRouter(modules.EdgeHubModuleName, modules.TwinGroup, "", ""), nil).
				Times(tt.receiveTimes)
			mockAdapter.EXPECT().Receive().
				Return(*model.NewMessage(""), errors.New("Connection Refused")).
				Times(1)
			go tt.hub.routeToEdge()
			stop := <-tt.hub.reconnectChan
			if stop != struct{}{} {
				t.Errorf("TestRouteToEdge error got: %v want: %v", stop, struct{}{})
			}
		})
	}
}

// TestSendToCloud() tests whether the send to cloud functionality works properly
func TestSendToCloud(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	mockAdapter := edgehub.NewMockAdapter(mockCtrl)

	msg := model.NewMessage("").BuildHeader("test_id", "", 1)
	msg.Header.Sync = true
	tests := []struct {
		name            string
		hub             *EdgeHub
		message         model.Message
		expectedError   string
		mockError       error
		HeartbeatPeriod int32
	}{
		{
			name: "send to cloud with proper input",
			hub: &EdgeHub{
				chClient: mockAdapter,
			},
			HeartbeatPeriod: 6,
			message:         *msg,
			expectedError:   "",
			mockError:       nil,
		},
		{
			name: "Wait Error in send to cloud",
			hub: &EdgeHub{
				chClient: mockAdapter,
			},
			HeartbeatPeriod: 3,
			message:         *msg,
			expectedError:   "",
			mockError:       nil,
		},
		{
			name: "Send Failure in send to cloud",
			hub: &EdgeHub{
				chClient: mockAdapter,
			},
			HeartbeatPeriod: 3,
			message:         model.Message{},
			expectedError:   "failed to send message, error: Connection Refused",
			mockError:       errors.New("Connection Refused"),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockAdapter.EXPECT().Send(gomock.Any()).Return(tt.mockError).Times(1)
			config.Config.Heartbeat = tt.HeartbeatPeriod
			err := tt.hub.sendToCloud(tt.message)
			if tt.expectedError != "" {
				require.ErrorContains(t, err, tt.expectedError)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

// TestRouteToCloud() tests the reception of the message from the beehive framework and forwarding of that message to cloud
func TestRouteToCloud(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	mockAdapter := edgehub.NewMockAdapter(mockCtrl)
	config.Config.MessageQPS = 3
	config.Config.MessageBurst = 6
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
			// Expect one send invoked by priority sender
			mockAdapter.EXPECT().Send(gomock.Any()).Return(nil).Times(1)

			// optional: register module (kept from historical test)
			core.Register(&EdgeHub{enable: true})
			go tt.hub.routeToCloud()
			go tt.hub.runPrioritySender()
			time.Sleep(100 * time.Millisecond)

			// enqueue one message to be routed to cloud
			msg := model.NewMessage("").BuildHeader("test_id", "", 1)
			beehiveContext.Send(modules.EdgeHubModuleName, *msg)

			// allow some time for sender to process
			time.Sleep(300 * time.Millisecond)

			// cleanup sender to avoid goroutine leak
			close(tt.hub.sendPQStop)
			tt.hub.sendPQ.Close()
		})
	}
}

// TestKeepalive() tests whether ping message sent to the cloud at regular intervals happens properly
func TestKeepalive(t *testing.T) {
	CertFile := "/tmp/kubeedge/certs/edge.crt"
	KeyFile := "/tmp/kubeedge/certs/edge.key"

	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	mockAdapter := edgehub.NewMockAdapter(mockCtrl)
	tests := []struct {
		name string
		hub  *EdgeHub
	}{
		{
			name: "Heartbeat failure Case",
			hub:  newEdgeHub(true),
		},
	}
	edgeHubConfig := config.Config
	edgeHubConfig.TLSCertFile = CertFile
	edgeHubConfig.TLSPrivateKeyFile = KeyFile
	config.Config.Heartbeat = 2

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// inject mock client
			tt.hub.chClient = mockAdapter
			// single keepalive send success
			mockAdapter.EXPECT().Send(gomock.Any()).Return(nil).Times(1)

			go tt.hub.keepalive()
			go tt.hub.runPrioritySender()
			time.Sleep(200 * time.Millisecond)

			// allow first send only (heartbeat=2s prevents a second send during test)
			time.Sleep(200 * time.Millisecond)

			// cleanup
			close(tt.hub.sendPQStop)
			tt.hub.sendPQ.Close()
		})
	}
}
