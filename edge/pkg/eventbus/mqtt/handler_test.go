/*
Copyright 2024 The KubeEdge Authors.

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

package mqtt

import (
	"encoding/base64"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kubeedge/beehive/pkg/common"
	beehiveContext "github.com/kubeedge/beehive/pkg/core/context"
	beehiveModel "github.com/kubeedge/beehive/pkg/core/model"
	messagepkg "github.com/kubeedge/kubeedge/edge/pkg/common/message"
	"github.com/kubeedge/kubeedge/edge/pkg/common/modules"
)

func TestHandleDeviceTwin(t *testing.T) {
	assert := assert.New(t)

	topic := "$hw/events/device/test-device/twin/update"
	payload := []byte("test payload")

	beehiveContext.InitContext([]string{common.MsgCtxTypeChannel})

	module := &common.ModuleInfo{
		ModuleName: modules.TwinGroup,
		ModuleType: common.MsgCtxTypeChannel,
	}
	beehiveContext.AddModule(module)
	beehiveContext.AddModuleGroup(module.ModuleName, module.ModuleName)

	want := beehiveModel.NewMessage("").
		BuildRouter(modules.BusGroup, modules.UserGroup,
			base64.URLEncoding.EncodeToString([]byte(topic)),
			messagepkg.OperationResponse).
		FillBody(string(payload))

	handleDeviceTwin(topic, payload)

	received, err := beehiveContext.Receive(modules.TwinGroup)
	assert.NoError(err)

	assert.Equal(want.GetSource(), received.GetSource())
	assert.Equal(want.GetGroup(), received.GetGroup())
	assert.Equal(want.GetResource(), received.GetResource())
	assert.Equal(want.GetOperation(), received.GetOperation())
	assert.Equal(want.GetContent(), received.GetContent())
}

func TestHandleUploadTopic(t *testing.T) {
	assert := assert.New(t)

	topic := "SYS/dis/upload_records"
	payload := []byte("test upload payload")

	beehiveContext.InitContext([]string{common.MsgCtxTypeChannel})

	module := &common.ModuleInfo{
		ModuleName: modules.HubGroup,
		ModuleType: common.MsgCtxTypeChannel,
	}
	beehiveContext.AddModule(module)
	beehiveContext.AddModuleGroup(module.ModuleName, module.ModuleName)

	want := beehiveModel.NewMessage("").
		BuildRouter(modules.BusGroup, modules.UserGroup,
			topic,
			beehiveModel.UploadOperation).
		FillBody(string(payload))

	handleUploadTopic(topic, payload)

	received, err := beehiveContext.Receive(modules.HubGroup)
	assert.NoError(err)

	assert.Equal(want.GetSource(), received.GetSource())
	assert.Equal(want.GetGroup(), received.GetGroup())
	assert.Equal(want.GetResource(), received.GetResource())
	assert.Equal(want.GetOperation(), received.GetOperation())
	assert.Equal(want.GetContent(), received.GetContent())
}
