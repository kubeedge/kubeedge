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

package eventbus

import (
	"errors"
	"os"
	"reflect"
	"testing"

	"github.com/agiledragon/gomonkey/v2"
	"github.com/stretchr/testify/assert"

	"github.com/kubeedge/api/apis/componentconfig/edgecore/v1alpha2"
	eventconfig "github.com/kubeedge/kubeedge/edge/pkg/eventbus/config"
	mqttBus "github.com/kubeedge/kubeedge/edge/pkg/eventbus/mqtt"
)

// exitCode is the value the patched os.Exit panics with.
type exitCode int

// TestStartExitsWhenExternalMqttClientFails checks that Start exits with status 1
// when the clients of the external MQTT broker cannot be created, for example
// because the eventBus TLS files cannot be loaded.
func TestStartExitsWhenExternalMqttClientFails(t *testing.T) {
	origMode := eventconfig.Config.MqttMode
	origHub := mqttBus.MQTTHub
	defer func() {
		eventconfig.Config.MqttMode = origMode
		mqttBus.MQTTHub = origHub
	}()
	eventconfig.Config.MqttMode = v1alpha2.MqttModeExternal

	patchRegister := gomonkey.ApplyFunc(mqttBus.RegisterMsgHandler, func() {})
	defer patchRegister.Reset()
	patchExit := gomonkey.ApplyFunc(os.Exit, func(code int) { panic(exitCode(code)) })
	defer patchExit.Reset()
	// If Start did not exit, it would go on to block in pubCloudMsgToEdge.
	patchPubCloudMsg := gomonkey.ApplyPrivateMethod(reflect.TypeOf(&eventbus{}), "pubCloudMsgToEdge",
		func(_ *eventbus) { panic("Start went on after the init error") })
	defer patchPubCloudMsg.Reset()

	hubType := reflect.TypeOf(&mqttBus.Client{})
	tests := []struct {
		name   string
		subErr error
		pubErr error
	}{
		{name: "sub client fails", subErr: errors.New("sub failed")},
		{name: "pub client fails", pubErr: errors.New("pub failed")},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			patchSub := gomonkey.ApplyMethod(hubType, "InitSubClient",
				func(_ *mqttBus.Client) error { return tt.subErr })
			defer patchSub.Reset()
			patchPub := gomonkey.ApplyMethod(hubType, "InitPubClient",
				func(_ *mqttBus.Client) error { return tt.pubErr })
			defer patchPub.Reset()

			assert.PanicsWithValue(t, exitCode(1), func() { (&eventbus{}).Start() })
		})
	}
}
