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

package authorization

import (
	"testing"

	"k8s.io/kubernetes/pkg/apis/authorization"

	"github.com/kubeedge/beehive/pkg/core/model"
	cloudhubmodel "github.com/kubeedge/kubeedge/cloud/pkg/cloudhub/common/model"
)

func TestGetBuiltinResourceAttributes(t *testing.T) {
	tests := []struct {
		name    string
		router  model.MessageRoute
		want    authorization.ResourceAttributes
		wantErr bool
	}{
		{
			name:   "secret query message",
			router: model.MessageRoute{Operation: model.QueryOperation, Resource: "ns/secret/sc"},
			want:   authorization.ResourceAttributes{Namespace: "ns", Name: "sc", Verb: "get", Group: "", Version: "v1", Resource: "secrets"},
		},
		{
			name:   "patch node message",
			router: model.MessageRoute{Operation: model.PatchOperation, Resource: "default/nodepatch/node"},
			want:   authorization.ResourceAttributes{Namespace: "", Name: "node", Verb: "patch", Group: "", Version: "v1", Resource: "nodes", Subresource: "status"},
		},
		{
			name:   "insert nodestatus message",
			router: model.MessageRoute{Operation: model.InsertOperation, Resource: "default/nodestatus/node"},
			want:   authorization.ResourceAttributes{Namespace: "", Name: "node", Verb: "create", Group: "", Version: "v1", Resource: "nodes"},
		},
		{
			name:   "insert podstatus message",
			router: model.MessageRoute{Operation: model.InsertOperation, Resource: "default/podstatus/pod"},
			want:   authorization.ResourceAttributes{Namespace: "default", Name: "pod", Verb: "create", Group: "", Version: "v1", Resource: "pods"},
		},
		{
			name:   "secret patch message",
			router: model.MessageRoute{Operation: model.DeleteOperation, Resource: "ns/secret/sc"},
			want:   authorization.ResourceAttributes{Namespace: "ns", Name: "sc", Verb: "delete", Group: "", Version: "v1", Resource: "secrets"},
		},
		{
			name:   "secret update message",
			router: model.MessageRoute{Operation: model.UpdateOperation, Resource: "ns/secret/sc"},
			want:   authorization.ResourceAttributes{Namespace: "ns", Name: "sc", Verb: "update", Group: "", Version: "v1", Resource: "secrets"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := getBuiltinResourceAttributes(tt.router)
			if err != nil {
				if !tt.wantErr {
					t.Errorf("getBuiltinResourceAttributes(): unexpect error: %v", err)
				}
				return
			}
			if tt.wantErr || *got != tt.want {
				t.Errorf("getBuiltinResourceAttributes() got = %v, want %v", *got, tt.want)
			}
		})
	}
}

func TestGetKubeedgeResourceAttributes(t *testing.T) {
	tests := []struct {
		name   string
		router model.MessageRoute
		want   authorization.NonResourceAttributes
	}{
		{
			name:   "keepalive message",
			router: model.MessageRoute{Operation: cloudhubmodel.OpKeepalive, Resource: "test"},
			want:   authorization.NonResourceAttributes{Verb: cloudhubmodel.OpKeepalive, Path: "test"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := getKubeedgeResourceAttributes(tt.router)
			if *got != tt.want {
				t.Errorf("getKubeedgeResourceAttributes() got = %v, want %v", *got, tt.want)
			}
		})
	}
}

func TestIsKubeedgeResourceMessage(t *testing.T) {
	tests := []struct {
		name   string
		router model.MessageRoute
		result bool
	}{
		{
			name:   "keepalive message",
			router: model.MessageRoute{Operation: cloudhubmodel.OpKeepalive},
			result: true,
		},
		{
			name:   "device twin message",
			router: model.MessageRoute{Source: cloudhubmodel.ResTwin},
			result: true,
		},
		{
			name:   "k8s ca message",
			router: model.MessageRoute{Resource: model.ResourceTypeK8sCA},
			result: true,
		},
		{
			name:   "rule status message",
			router: model.MessageRoute{Resource: "ns/rulestatus/rs"},
			result: true,
		},
		{
			name:   "device twin message",
			router: model.MessageRoute{Source: cloudhubmodel.ResTwin},
			result: true,
		},
		{
			name:   "configmap message",
			router: model.MessageRoute{Operation: model.QueryOperation, Resource: "ns/configmap/test-cm", Source: "edged", Group: "meta"},
			result: false,
		},
		{
			name:   "podstatus list message",
			router: model.MessageRoute{Operation: model.UpdateOperation, Resource: "ns/podstatus"},
			result: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isKubeedgeResourceMessage(tt.router)
			if got != tt.result {
				t.Errorf("isKubeedgeResourceMessage() got = %v, want %v", got, tt.result)
			}
		})
	}
}

func TestGetAuthorizerAttributes(t *testing.T) {
	tests := []struct {
		name              string
		router            model.MessageRoute
		hubInfo           cloudhubmodel.HubInfo
		wantErr           bool
		isKubeedgeMessage bool
	}{
		{
			name:    "invalid message",
			router:  model.MessageRoute{},
			wantErr: true,
		},
		{
			name:              "device twin message",
			router:            model.MessageRoute{Source: cloudhubmodel.ResTwin},
			isKubeedgeMessage: true,
		},
		{
			name:   "configmap message",
			router: model.MessageRoute{Operation: model.QueryOperation, Resource: "ns/configmap/test-cm", Source: "edged", Group: "meta"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := getAuthorizerAttributes(tt.router, tt.hubInfo)
			if err != nil {
				if !tt.wantErr {
					t.Errorf("getAuthorizerAttributes(): unexpect error: %v", err)
				}
				return
			}

			if isKubeedgeResourceAttributes(got) != tt.isKubeedgeMessage {
				t.Errorf("getAuthorizerAttributes() got = %v, want %v", got, tt.isKubeedgeMessage)
			}
		})
	}
}
