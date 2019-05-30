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

package clients

import (
	"net/http"
	"reflect"
	"testing"
	"time"

	"github.com/kubeedge/kubeedge/edge/pkg/edgehub/clients/wsclient"
	"github.com/kubeedge/kubeedge/edge/pkg/edgehub/config"
)

//TestGetClient() tests whether the webSocket client is returned properly
func TestGetClient(t *testing.T) {
	type args struct {
		clientType string
		config     *config.EdgeHubConfig
	}
	tests := []struct {
		name string
		args args
		want Adapter
		err  error
	}{
		{"TestGetClient: Positive Test Case", args{
			clientType: ClientTypeWebSocket,
			config: &config.EdgeHubConfig{
				WSConfig: config.WebSocketConfig{
					URL:              "ws://127.0.0.1:20000/fake_group_id/events",
					CertFilePath:     "/tmp/edge.crt",
					KeyFilePath:      "/tmp/edge.key",
					HandshakeTimeout: 500 * time.Second,
					WriteDeadline:    100 * time.Second,
					ReadDeadline:     100 * time.Second,
					ExtendHeader:     http.Header{},
				},
			},
		}, wsclient.NewWebSocketClient(&wsclient.WebSocketConfig{
			URL:              "ws://127.0.0.1:20000/fake_group_id/events",
			CertFilePath:     "/tmp/edge.crt",
			KeyFilePath:      "/tmp/edge.key",
			HandshakeTimeout: 500 * time.Second,
			WriteDeadline:    100 * time.Second,
			ReadDeadline:     100 * time.Second,
			ExtendHeader:     http.Header{},
		}),
			nil,
		},

		{"TestGetClient: Negative Test Case", args{
			clientType: "WrongClientType",
		}, nil, ErrorWrongClientType},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got, err := GetClient(tt.args.clientType, tt.args.config); !reflect.DeepEqual(got, tt.want) || err != tt.err {
				t.Errorf("GetClient() = %v, want %v", got, tt.want)
			}
		})
	}
}
