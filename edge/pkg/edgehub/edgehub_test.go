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
	"testing"

	"github.com/kubeedge/kubeedge/edge/pkg/edgehub/config"
	"github.com/kubeedge/kubeedge/pkg/apis/componentconfig/edgecore/v1alpha2"
)

func TestGetCertSyncChannel(t *testing.T) {
	t.Run("GetCertSyncChannel()", func(t *testing.T) {
		certSync := GetCertSyncChannel()
		if certSync != nil {
			t.Errorf("GetCertSyncChannel() returned unexpected result. got = %v, want = %v", certSync, nil)
		}
	})
}

func TestNewCertSyncChannel(t *testing.T) {
	t.Run("NewCertSyncChannel()", func(t *testing.T) {
		certSync := NewCertSyncChannel()
		if len(certSync) != 2 {
			t.Errorf("NewCertSyncChannel() returned  unexpected results. size got = %d, size want = 2", len(certSync))
		}
		if _, ok := certSync["edgestream"]; !ok {
			t.Error("NewCertSyncChannel() returned  unexpected results. expected key edgestream to be present but it was not available.")
		}
		if _, ok := certSync["metamanager"]; !ok {
			t.Error("NewCertSyncChannel() returned  unexpected results. expected key metamanager to be present but it was not available.")
		}
	})
}

func TestRegister(t *testing.T) {
	tests := []struct {
		eh           *v1alpha2.EdgeHub
		nodeName     string
		name         string
		wantNodeName string
	}{
		{
			name:         "",
			nodeName:     "test1",
			wantNodeName: "test1",
			eh:           &v1alpha2.EdgeHub{WebSocket: &v1alpha2.EdgeHubWebSocket{Server: "localhost:8080"}, ProjectID: "test_id"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			Register(tt.eh, tt.nodeName)

			if config.Config.NodeName != tt.wantNodeName {
				t.Errorf("failed to Register(). Nodename : got = %s, want = %s", config.Config.NodeName, tt.wantNodeName)
			}
		})
	}
}

func TestName(t *testing.T) {
	t.Run("EdgeHub.Name()", func(t *testing.T) {
		if got := (&EdgeHub{}).Name(); got != "websocket" {
			t.Errorf("EdgeHub.Name() returned unexpected result. got = %s, want = websocket", got)
		}
	})
}

func TestGroup(t *testing.T) {
	t.Run("EdgeHub.Group()", func(t *testing.T) {
		if got := (&EdgeHub{}).Group(); got != "hub" {
			t.Errorf("EdgeHub.Group() returned unexpected result. got = %s, want = hub", got)
		}
	})
}

func TestEnable(t *testing.T) {
	tests := []struct {
		eh   *EdgeHub
		want bool
		name string
	}{
		{
			name: "Enable true",
			want: true,
			eh:   &EdgeHub{enable: true},
		},
		{
			name: "Enable false",
			want: false,
			eh:   &EdgeHub{enable: false},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.eh.Enable(); got != tt.want {
				t.Errorf("EdgeHub.Enable() returned expected results. got = %v, want = %v", got, tt.want)
			}
		})
	}
}
