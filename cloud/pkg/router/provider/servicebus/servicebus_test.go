/*
Copyright 2026 The KubeEdge Authors.

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

package servicebus

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kubeedge/beehive/pkg/common"
	beehiveContext "github.com/kubeedge/beehive/pkg/core/context"
	"github.com/kubeedge/beehive/pkg/core/model"
	"github.com/kubeedge/kubeedge/cloud/pkg/common/modules"
	"github.com/kubeedge/kubeedge/cloud/pkg/router/listener"
)

func TestUnregisterListenerUndoesRegisterListener(t *testing.T) {
	beehiveContext.InitContext([]string{common.MsgCtxTypeChannel})
	beehiveContext.AddModule(&common.ModuleInfo{
		ModuleName: modules.CloudHubModuleName,
		ModuleType: common.MsgCtxTypeChannel,
	})

	sb := &ServiceBus{nodeName: "edge-node", TargetURL: "http://127.0.0.1:9000/echo"}
	upload := model.NewMessage("").BuildRouter("servicebus", modules.UserGroup,
		"node/edge-node/http://127.0.0.1:9000/echo", model.UploadOperation)

	assert.NoError(t, sb.RegisterListener(func(interface{}) (interface{}, error) { return nil, nil }))
	start, err := beehiveContext.Receive(modules.CloudHubModuleName)
	assert.NoError(t, err)
	assert.Equal(t, "node/edge-node/http://127.0.0.1:9000/echo", start.GetResource())
	assert.NoError(t, listener.MessageHandlerInstance.HandleMessage(upload))

	sb.UnregisterListener()
	stop, err := beehiveContext.Receive(modules.CloudHubModuleName)
	assert.NoError(t, err)
	assert.Equal(t, "stop", stop.GetOperation())
	assert.Equal(t, start.GetResource(), stop.GetResource())
	assert.Error(t, listener.MessageHandlerInstance.HandleMessage(upload))
}
