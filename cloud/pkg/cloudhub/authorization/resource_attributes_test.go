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

	"k8s.io/apiserver/pkg/authentication/user"
	"k8s.io/apiserver/pkg/authorization/authorizer"
	"k8s.io/kubernetes/pkg/apis/authorization"

	"github.com/kubeedge/beehive/pkg/core/model"
	cloudhubmodel "github.com/kubeedge/kubeedge/cloud/pkg/cloudhub/common/model"
	taskutil "github.com/kubeedge/kubeedge/cloud/pkg/taskmanager/v1alpha1/util"
	"github.com/kubeedge/kubeedge/pkg/metaserver"
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
		{
			name:    "empty resource string",
			router:  model.MessageRoute{Operation: model.QueryOperation, Resource: ""},
			wantErr: true,
		},
		{
			name:    "malformed resource - only one segment",
			router:  model.MessageRoute{Operation: model.QueryOperation, Resource: "onlyone"},
			wantErr: true,
		},
		{
			name:    "malformed resource - too many segments",
			router:  model.MessageRoute{Operation: model.QueryOperation, Resource: "a/b/c/d"},
			wantErr: true,
		},
		{
			name:    "malformed resource - empty resourceType",
			router:  model.MessageRoute{Operation: model.QueryOperation, Resource: "ns//name"},
			wantErr: true,
		},
		{
			name:    "unknown resource type",
			router:  model.MessageRoute{Operation: model.QueryOperation, Resource: "ns/unknownresource/name"},
			wantErr: true,
		},
		{
			name:    "unknown operation",
			router:  model.MessageRoute{Operation: "unknownop", Resource: "ns/secret/sc"},
			wantErr: true,
		},
		{
			name:   "query serviceaccounttoken - verb must be create",
			router: model.MessageRoute{Operation: model.QueryOperation, Resource: "ns/serviceaccounttoken/sa"},
			want:   authorization.ResourceAttributes{Namespace: "ns", Name: "sa", Verb: "create", Group: "", Version: "v1", Resource: "serviceaccounts", Subresource: "token"},
		},
		{
			name:   "query node - non-namespaced, namespace stripped",
			router: model.MessageRoute{Operation: model.QueryOperation, Resource: "default/node/mynode"},
			want:   authorization.ResourceAttributes{Namespace: "", Name: "mynode", Verb: "get", Group: "", Version: "v1", Resource: "nodes"},
		},
		{
			name:   "query pod - namespaced",
			router: model.MessageRoute{Operation: model.QueryOperation, Resource: "default/pod/mypod"},
			want:   authorization.ResourceAttributes{Namespace: "default", Name: "mypod", Verb: "get", Group: "", Version: "v1", Resource: "pods"},
		},
		{
			name:   "insert pod (not podstatus) - stays as pod",
			router: model.MessageRoute{Operation: model.InsertOperation, Resource: "default/pod/mypod"},
			want:   authorization.ResourceAttributes{Namespace: "default", Name: "mypod", Verb: "create", Group: "", Version: "v1", Resource: "pods"},
		},
		{
			name:   "patch podpatch",
			router: model.MessageRoute{Operation: model.PatchOperation, Resource: "default/podpatch/mypod"},
			want:   authorization.ResourceAttributes{Namespace: "default", Name: "mypod", Verb: "patch", Group: "", Version: "v1", Resource: "pods", Subresource: "status"},
		},
		{
			name:   "two-segment resource (no resourceName)",
			router: model.MessageRoute{Operation: model.QueryOperation, Resource: "ns/configmap"},
			want:   authorization.ResourceAttributes{Namespace: "ns", Name: "", Verb: "get", Group: "", Version: "v1", Resource: "configmaps"},
		},
		{
			name:   "query lease",
			router: model.MessageRoute{Operation: model.QueryOperation, Resource: "ns/lease/my-lease"},
			want:   authorization.ResourceAttributes{Namespace: "ns", Name: "my-lease", Verb: "get", Group: "coordination.k8s.io", Version: "v1", Resource: "leases"},
		},
		{
			name:    "query CSR - non-namespaced",
			router:  model.MessageRoute{Operation: model.QueryOperation, Resource: "default/certificatesigningrequests/mycsr"},
			wantErr: true,
		},
		{
			name:   "update nodestatus",
			router: model.MessageRoute{Operation: model.UpdateOperation, Resource: "default/nodestatus/mynode"},
			want:   authorization.ResourceAttributes{Namespace: "", Name: "mynode", Verb: "update", Group: "", Version: "v1", Resource: "nodes", Subresource: "status"},
		},
		{
			name:   "delete pod",
			router: model.MessageRoute{Operation: model.DeleteOperation, Resource: "default/pod/mypod"},
			want:   authorization.ResourceAttributes{Namespace: "default", Name: "mypod", Verb: "delete", Group: "", Version: "v1", Resource: "pods"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := getBuiltinResourceAttributes(tt.router)
			if err != nil {
				if !tt.wantErr {
					t.Errorf("getBuiltinResourceAttributes(): unexpected error: %v", err)
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
		{
			name:   "response operation",
			router: model.MessageRoute{Operation: model.ResponseOperation},
			result: true,
		},
		{
			name:   "response error operation",
			router: model.MessageRoute{Operation: model.ResponseErrorOperation},
			result: true,
		},
		{
			name:   "upload operation",
			router: model.MessageRoute{Operation: model.UploadOperation},
			result: true,
		},
		{
			name:   "task prepull operation",
			router: model.MessageRoute{Operation: taskutil.TaskPrePull},
			result: true,
		},
		{
			name:   "task upgrade operation",
			router: model.MessageRoute{Operation: taskutil.TaskUpgrade},
			result: true,
		},
		{
			name:   "metaserver source",
			router: model.MessageRoute{Source: metaserver.MetaServerSource},
			result: true,
		},
		{
			name:   "volume resource",
			router: model.MessageRoute{Resource: "ns/volume/pv1"},
			result: true,
		},
		{
			name:   "podstatus with name - not kubeedge resource",
			router: model.MessageRoute{Operation: model.UpdateOperation, Resource: "ns/podstatus/somepod"},
			result: false,
		},
		{
			name:   "non-matching operation and source",
			router: model.MessageRoute{Operation: model.InsertOperation, Source: "edged", Resource: "ns/pod/mypod"},
			result: false,
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
		{
			name:    "getBuiltinResourceAttributes error - malformed resource",
			router:  model.MessageRoute{Operation: model.QueryOperation, Resource: "onlyone"},
			wantErr: true,
		},
		{
			name:    "getBuiltinResourceAttributes error - unknown resource type",
			router:  model.MessageRoute{Operation: model.QueryOperation, Resource: "ns/unknowntype/name"},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := getAuthorizerAttributes(tt.router, tt.hubInfo)
			if err != nil {
				if !tt.wantErr {
					t.Errorf("getAuthorizerAttributes(): unexpected error: %v", err)
				}
				return
			}

			if isKubeedgeResourceAttributes(got) != tt.isKubeedgeMessage {
				t.Errorf("getAuthorizerAttributes() got = %v, want %v", got, tt.isKubeedgeMessage)
			}
		})
	}
}

func TestParseResourceStrict(t *testing.T) {
	tests := []struct {
		name        string
		resource    string
		wantNS      string
		wantResType string
		wantResName string
		wantErr     bool
	}{
		{
			name:     "empty string",
			resource: "",
			wantErr:  true,
		},
		{
			name:     "single segment - too few",
			resource: "onlyone",
			wantErr:  true,
		},
		{
			name:     "four segments - too many",
			resource: "a/b/c/d",
			wantErr:  true,
		},
		{
			name:     "empty resourceType",
			resource: "ns//name",
			wantErr:  true,
		},
		{
			name:        "two segments - no resourceName",
			resource:    "ns/configmap",
			wantNS:      "ns",
			wantResType: "configmap",
			wantResName: "",
		},
		{
			name:        "three segments - with resourceName",
			resource:    "ns/configmap/mycm",
			wantNS:      "ns",
			wantResType: "configmap",
			wantResName: "mycm",
		},
		{
			name:        "empty namespace is allowed",
			resource:    "/pod/mypod",
			wantNS:      "",
			wantResType: "pod",
			wantResName: "mypod",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ns, resType, resName, err := parseResourceStrict(tt.resource)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseResourceStrict(%q) error = %v, wantErr %v", tt.resource, err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if ns != tt.wantNS || resType != tt.wantResType || resName != tt.wantResName {
					t.Errorf("parseResourceStrict(%q) = (%q, %q, %q), want (%q, %q, %q)",
						tt.resource, ns, resType, resName, tt.wantNS, tt.wantResType, tt.wantResName)
				}
			}
		})
	}
}

func TestIsKubeedgeResourceAttributes(t *testing.T) {
	tests := []struct {
		name   string
		attrs  authorizer.Attributes
		result bool
	}{
		{
			name:   "nil attrs",
			attrs:  nil,
			result: false,
		},
		{
			name:   "attrs with nil user",
			attrs:  &fakeAttrs{AttributesRecord: authorizer.AttributesRecord{User: nil}},
			result: false,
		},
		{
			name:   "attrs without kubeedge key",
			attrs:  &fakeAttrs{AttributesRecord: authorizer.AttributesRecord{User: &fakeUser{DefaultInfo: user.DefaultInfo{}}}},
			result: false,
		},
		{
			name:   "attrs with kubeedge key",
			attrs:  &fakeAttrs{AttributesRecord: authorizer.AttributesRecord{User: &fakeUser{DefaultInfo: user.DefaultInfo{Extra: map[string][]string{kubeedgeResourceKey: {}}}}}},
			result: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isKubeedgeResourceAttributes(tt.attrs)
			if got != tt.result {
				t.Errorf("isKubeedgeResourceAttributes() = %v, want %v", got, tt.result)
			}
		})
	}
}

type fakeUser struct {
	user.DefaultInfo
}

type fakeAttrs struct {
	authorizer.AttributesRecord
}
