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

package dtmanager

import (
	"errors"
	"reflect"
	"testing"
	"time"

	"github.com/kubeedge/beehive/pkg/core/context"
	"github.com/kubeedge/beehive/pkg/core/model"
	cloudconn "github.com/kubeedge/kubeedge/edge/pkg/common/cloudconnection"
	"github.com/kubeedge/kubeedge/edge/pkg/devicetwin/dtcommon"
	"github.com/kubeedge/kubeedge/edge/pkg/devicetwin/dtcontext"
	"github.com/kubeedge/kubeedge/edge/pkg/devicetwin/dttype"
)

// TestStartAction is function to test Start() when value is passed in ReceiverChan.
func TestStartAction(t *testing.T) {
	mainContext := context.GetContext(context.MsgCtxTypeChannel)
	dtContextStateConnected, _ := dtcontext.InitDTContext(mainContext)
	dtContextStateConnected.State = dtcommon.Connected
	receiveChanActionPresent := make(chan interface{}, 1)
	receiveChanActionPresent <- &dttype.DTMessage{Action: dtcommon.SendToCloud, Identity: "identity", Msg: &model.Message{Header: model.MessageHeader{ID: "message"}, Content: "msg"}}
	receiveChanActionNotPresent := make(chan interface{}, 1)
	receiveChanActionNotPresent <- &dttype.DTMessage{Action: "action", Identity: "identity", Msg: &model.Message{Content: "msg"}}
	tests := []struct {
		name   string
		Worker Worker
	}{
		{
			name: "StartTest-ActionPresentInActionCallback",
			Worker: Worker{
				ReceiverChan: receiveChanActionPresent,
				DTContexts:   dtContextStateConnected,
			},
		},
		{
			name: "StartTest-ActionNotPresentInActionCallback",
			Worker: Worker{
				ReceiverChan: receiveChanActionNotPresent,
				DTContexts:   dtContextStateConnected,
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			cw := CommWorker{
				Worker: test.Worker,
			}
			go cw.Start()
			time.Sleep(1 * time.Millisecond)
			if test.Worker.ReceiverChan == receiveChanActionPresent {
				_, exist := test.Worker.DTContexts.ConfirmMap.Load("message")
				if !exist {
					t.Errorf("Start Failed to store message in ConfirmMap")
				}
			}
		})
	}
}

// TestStartHeartBeat is function to test Start() when value is passed in HeartBeatChan.
func TestStartHeartBeat(t *testing.T) {
	mainContext := context.GetContext(context.MsgCtxTypeChannel)
	dtContexts, _ := dtcontext.InitDTContext(mainContext)
	heartChanStop := make(chan interface{}, 1)
	heartChanPing := make(chan interface{}, 1)
	heartChanStop <- "stop"
	heartChanPing <- "ping"
	tests := []struct {
		name   string
		Worker Worker
		Group  string
	}{
		{
			name: "StartTest-PingInHeartBeatChannel",
			Worker: Worker{
				HeartBeatChan: heartChanPing,
				DTContexts:    dtContexts,
			},
			Group: "group",
		},
		{
			name: "StartTest-StopInHeartBeatChannel",
			Worker: Worker{
				HeartBeatChan: heartChanStop,
				DTContexts:    dtContexts,
			},
			Group: "group",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			cw := CommWorker{
				Worker: test.Worker,
				Group:  test.Group,
			}
			go cw.Start()
			time.Sleep(1 * time.Millisecond)
			if test.Worker.HeartBeatChan == heartChanPing {
				_, exist := test.Worker.DTContexts.ModulesHealth.Load("group")
				if !exist {
					t.Errorf("Start Failed to add module in context")
				}
			}
		})
	}
}

// TestDealSendToEdge is function to test dealsendToedge().
func TestDealSendToEdge(t *testing.T) {
	mainContext := context.GetContext(context.MsgCtxTypeChannel)
	dtContexts, _ := dtcontext.InitDTContext(mainContext)
	tests := []struct {
		name     string
		context  *dtcontext.DTContext
		resource string
		msg      interface{}
	}{
		{
			name:    "dealSendToEdgeTest",
			context: dtContexts,
			msg:     &model.Message{},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			dealSendToEdge(test.context, test.resource, test.msg)
		})
	}
}

// TestDealSendToCloud is function to test dealSendToCloud().
func TestDealSendToCloud(t *testing.T) {
	mainContext := context.GetContext(context.MsgCtxTypeChannel)
	dtContextStateDisconnected, _ := dtcontext.InitDTContext(mainContext)
	dtContextStateConnected, _ := dtcontext.InitDTContext(mainContext)
	dtContextStateConnected.State = dtcommon.Connected
	msg := &model.Message{Header: model.MessageHeader{ID: "message"}}
	expectedMessage := &dttype.DTMessage{Msg: msg, Action: dtcommon.SendToCloud, Type: dtcommon.CommModule}
	tests := []struct {
		name     string
		context  *dtcontext.DTContext
		resource string
		msg      interface{}
		wantErr  error
	}{
		{
			name:    "dealSendToCloudTest-StateDisconnected",
			context: dtContextStateDisconnected,
			msg:     "",
			wantErr: nil,
		},
		{
			name:    "dealSendToCloudTest-StateConnected",
			context: dtContextStateConnected,
			msg:     "",
			wantErr: errors.New("msg not Message type"),
		},
		{
			name:    "dealSendToCloudTest-ActualMsg",
			context: dtContextStateConnected,
			msg:     msg,
			wantErr: nil,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := dealSendToCloud(test.context, test.resource, test.msg)
			if !reflect.DeepEqual(err, test.wantErr) {
				t.Errorf("dealSendToCloud() error = %v, wantErr %v", err, test.wantErr)
				return
			}
			// Testing whether the message is properly stored in ConfirmMap of context when correct message is passed
			if err == nil && test.context.State == dtcommon.Connected {
				gotMsg, exist := test.context.ConfirmMap.Load("message")
				if !exist {
					t.Errorf("dealSendToCloud() failed to store message in ConfirmMap")
					return
				} else {
					if !reflect.DeepEqual(expectedMessage, gotMsg) {
						t.Errorf("dealSendToCloud() failed due to wrong gotMsg in ConfirmMap Got =%v Want=%v", test.msg, gotMsg)
					}
				}
			}
		})
	}
}

// TestDealLifeCycle is function to test dealLifeCycle().
func TestDealLifeCycle(t *testing.T) {
	mainContext := context.GetContext(context.MsgCtxTypeChannel)
	dtContext, _ := dtcontext.InitDTContext(mainContext)
	tests := []struct {
		name     string
		context  *dtcontext.DTContext
		resource string
		msg      interface{}
		wantErr  error
	}{
		{
			name:    "dealLifeCycleTest-WrongMessageFormat",
			context: dtContext,
			msg:     "",
			wantErr: errors.New("msg not Message type"),
		},
		{
			name:    "dealLifeCycleTest-CloudConnected",
			context: dtContext,
			msg:     &model.Message{Content: cloudconn.CloudConnected},
			wantErr: nil,
		},
		{
			name:    "dealLifeCycleTest-CloudNotConnected",
			context: dtContext,
			msg:     &model.Message{Content: cloudconn.CloudDisconnected},
			wantErr: nil,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := dealLifeCycle(test.context, test.resource, test.msg)
			if !reflect.DeepEqual(err, test.wantErr) {
				t.Errorf("dealLifeCycle() error = %v, wantErr %v", err, test.wantErr)
			}
		})
	}
}

// TestDealConfirm is function to test dealConfirm().
func TestDealConfirm(t *testing.T) {
	mainContext := context.GetContext(context.MsgCtxTypeChannel)
	dtContext, _ := dtcontext.InitDTContext(mainContext)
	tests := []struct {
		name     string
		context  *dtcontext.DTContext
		resource string
		msg      interface{}
		wantErr  error
	}{
		{
			name:    "dealConfirmTest-WrongMsg",
			context: dtContext,
			msg:     "",
			wantErr: errors.New("CommModule deal confirm, type not correct"),
		},
		{
			name:    "dealConfirmTest-CorrectMsg",
			context: dtContext,
			msg:     &model.Message{Header: model.MessageHeader{ID: "id", ParentID: "parentId"}},
			wantErr: nil,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := dealConfirm(test.context, test.resource, test.msg)
			if !reflect.DeepEqual(err, test.wantErr) {
				t.Errorf("dealConfirm() error = %v, wantErr %v", err, test.wantErr)
				return
			}
			if err == nil {
				_, exist := test.context.ConfirmMap.Load("parentId")
				if exist {
					t.Errorf("dealConfirm failed() ParentMessageId still present in context ConfirmMap")
				}
			}
		})
	}
}

// TestCheckConfirm is function to test checkConfirm().
func TestCheckConfirm(t *testing.T) {
	mainContext := context.GetContext(context.MsgCtxTypeChannel)
	dtContext, _ := dtcontext.InitDTContext(mainContext)
	dtContext.State = dtcommon.Connected
	dtContext.ConfirmMap.Store("emptyMessage", &dttype.DTMessage{})
	dtContext.ConfirmMap.Store("actionMessage", &dttype.DTMessage{Msg: &model.Message{}, Action: dtcommon.SendToCloud})
	tests := []struct {
		name    string
		Worker  Worker
		context *dtcontext.DTContext
		msg     interface{}
	}{
		{
			name:    "checkConfirmTest",
			Worker:  Worker{DTContexts: dtContext},
			context: dtContext,
			msg:     &dttype.DTMessage{Action: "action"},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			cw := CommWorker{
				Worker: test.Worker,
			}
			cw.checkConfirm(test.context, test.msg)
			_, exist := test.context.ConfirmMap.Load("actionMessage")
			if !exist {
				t.Errorf(" checkconfirm() failed because dealSendToCloud() failed to store message in ConfirmMap")
				return
			}
		})
	}
}
