/*
Copyright 2025 The KubeEdge Authors.

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

package dmiserver

import (
	"encoding/base64"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	pb "github.com/kubeedge/api/apis/dmi/v1beta1"
	"github.com/kubeedge/beehive/pkg/common"
	beehiveContext "github.com/kubeedge/beehive/pkg/core/context"
	messagepkg "github.com/kubeedge/kubeedge/edge/pkg/common/message"
	"github.com/kubeedge/kubeedge/edge/pkg/common/modules"
	"github.com/kubeedge/kubeedge/edge/pkg/devicetwin/dtcommon"
	pkgutil "github.com/kubeedge/kubeedge/pkg/util"
)

// initBeehiveForTwin registers modules.TwinGroup in a fresh channel context
// so that beehiveContext.SendToGroup(modules.TwinGroup, ...) delivers messages
// to a Receive-able channel in tests.
func initBeehiveForTwin(t *testing.T) {
	t.Helper()
	beehiveContext.InitContext([]string{common.MsgCtxTypeChannel})
	module := &common.ModuleInfo{
		ModuleName: modules.TwinGroup,
		ModuleType: common.MsgCtxTypeChannel,
	}
	beehiveContext.AddModule(module)
	beehiveContext.AddModuleGroup(module.ModuleName, module.ModuleName)
}

func TestHandleDeviceState(t *testing.T) {
	assert := assert.New(t)
	initBeehiveForTwin(t)

	req := &pb.ReportDeviceStatesRequest{
		DeviceName:      "device1",
		DeviceNamespace: "default",
		State:           "online",
	}
	payload := []byte(`{"state":"online"}`)

	// Build expected topic and base64-encoded resource
	deviceID := pkgutil.GetResourceID("default", "device1")
	topic := dtcommon.DeviceETPrefix + deviceID + dtcommon.DeviceETStateUpdateSuffix
	expectedResource := base64.URLEncoding.EncodeToString([]byte(topic))

	handleDeviceState(req, payload)

	// Use a bounded timeout so a regression fails promptly instead of hanging.
	done := make(chan struct{})
	go func() {
		defer close(done)
		msg, err := beehiveContext.Receive(modules.TwinGroup)
		if !assert.NoError(err) {
			return
		}
		// Verify complete route: source=bus, group=user, operation=response
		assert.Equal(modules.BusGroup, msg.GetSource())
		assert.Equal(modules.UserGroup, msg.GetGroup())
		assert.Equal(messagepkg.OperationResponse, msg.GetOperation())
		// Verify Base64-encoded resource topic
		assert.Equal(expectedResource, msg.GetResource())
		// Verify exact payload
		assert.Equal(string(payload), msg.GetContent())
	}()

	select {
	case <-done:
	case <-time.After(3 * time.Second):
		t.Fatal("timed out waiting for message from handleDeviceState")
	}
}

func TestHandleDeviceTwin(t *testing.T) {
	assert := assert.New(t)
	initBeehiveForTwin(t)

	req := &pb.ReportDeviceStatusRequest{
		DeviceName:      "device1",
		DeviceNamespace: "default",
	}
	payload := []byte(`{"twin":{}}`)

	// Build expected topic and base64-encoded resource
	deviceID := pkgutil.GetResourceID("default", "device1")
	topic := dtcommon.DeviceETPrefix + deviceID + dtcommon.TwinETUpdateSuffix
	expectedResource := base64.URLEncoding.EncodeToString([]byte(topic))

	handleDeviceTwin(req, payload)

	done := make(chan struct{})
	go func() {
		defer close(done)
		msg, err := beehiveContext.Receive(modules.TwinGroup)
		if !assert.NoError(err) {
			return
		}
		// Verify complete route: source=bus, group=user, operation=response
		assert.Equal(modules.BusGroup, msg.GetSource())
		assert.Equal(modules.UserGroup, msg.GetGroup())
		assert.Equal(messagepkg.OperationResponse, msg.GetOperation())
		// Verify Base64-encoded resource topic
		assert.Equal(expectedResource, msg.GetResource())
		// Verify exact payload
		assert.Equal(string(payload), msg.GetContent())
	}()

	select {
	case <-done:
	case <-time.After(3 * time.Second):
		t.Fatal("timed out waiting for message from handleDeviceTwin")
	}
}
