/*
Copyright 2023 The KubeEdge Authors.

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
	"time"

	"github.com/kubeedge/beehive/pkg/common"
	beehiveContext "github.com/kubeedge/beehive/pkg/core/context"
	"github.com/kubeedge/beehive/pkg/core/model"
	"github.com/kubeedge/kubeedge/edge/pkg/devicetwin/dtcommon"
	"github.com/kubeedge/kubeedge/edge/pkg/devicetwin/dtcontext"
	"github.com/kubeedge/kubeedge/edge/pkg/devicetwin/dttype"
)

const (
	Delay            = 10 * time.Millisecond
	MaxRetries       = 5
	Identity         = "identity"
	Message          = "message"
	Msg              = "msg"
	Action           = "action"
	TestAction       = "testAction"
	ActionPresent    = "StartTest-ActionPresentInActionCallback"
	ActionNotPresent = "StartTest-ActionNotPresentInActionCallback"
	PingHeartBeat    = "StartTest-PingInHeartBeatChannel"
	StopHeartBeat    = "StartTest-StopInHeartBeatChannel"
	Group            = "group"
)

// CaseWorkerStr is case struct of worker
type CaseWorkerStr struct {
	name   string
	Worker Worker
}

// CaseHeartBeatWorkerStr is case struct of worker for heartbeat
type CaseHeartBeatWorkerStr struct {
	name   string
	Worker Worker
	Group  string
}

type CasesMsgWorkerStr []struct {
	name     string
	context  *dtcontext.DTContext
	resource string
	msg      interface{}
	wantErr  error
}

// GenerateReceiveChanAction generates receive channel action
func GenerateReceiveChanAction(action, identity, id, content string) chan interface{} {
	channel := make(chan interface{}, 1)
	channel <- &dttype.DTMessage{
		Action:   action,
		Identity: identity,
		Msg: &model.Message{
			Header: model.MessageHeader{
				ID: id,
			},
			Content: content,
		},
	}
	return channel
}

// GenerateStartActionCase generates start action case
func GenerateStartActionCase(name string, channelPresent chan interface{}) CaseWorkerStr {
	dtContextStateConnected, _ := dtcontext.InitDTContext()
	dtContextStateConnected.State = dtcommon.Connected
	return CaseWorkerStr{
		name: name,
		Worker: Worker{
			ReceiverChan: channelPresent,
			DTContexts:   dtContextStateConnected,
		},
	}
}

// GenerateHeartBeatCase generates heart beat action case
func GenerateHeartBeatCase(name, group string, channel chan interface{}) CaseHeartBeatWorkerStr {
	beehiveContext.InitContext([]string{common.MsgCtxTypeChannel})

	dtContexts, _ := dtcontext.InitDTContext()
	return CaseHeartBeatWorkerStr{
		name: name,
		Worker: Worker{
			ReceiverChan: channel,
			DTContexts:   dtContexts,
		},
		Group: group,
	}
}

func generateMessageAttributes() map[string]*dttype.MsgAttr {
	messageAttributes := make(map[string]*dttype.MsgAttr)
	optional := true
	msgattr := &dttype.MsgAttr{
		Value:    "ON",
		Optional: &optional,
		Metadata: &dttype.TypeMetadata{
			Type: "device",
		},
	}
	messageAttributes["DeviceA"] = msgattr
	return messageAttributes
}
