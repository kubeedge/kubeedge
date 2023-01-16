/*
Copyright 2022 The KubeEdge Authors.

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

package config

import (
	"testing"

	"github.com/kubeedge/kubeedge/pkg/apis/componentconfig/edgecore/v1alpha2"
)

func TestInitConfigure(t *testing.T) {
	t.Run("configure only once", func(t *testing.T) {
		eh := &v1alpha2.EdgeHub{
			WebSocket: &v1alpha2.EdgeHubWebSocket{
				Server: "test_server",
			},
			ProjectID: "e632aba927ea4ac2b575ec1603d56f10",
		}
		InitConfigure(eh, "test 1")
		InitConfigure(eh, "test 2")
		InitConfigure(eh, "test 3")

		if Config.NodeName != "test 1" {
			t.Fatalf("InitConfigure() changes the config for each function calls")
		}
	})
}
