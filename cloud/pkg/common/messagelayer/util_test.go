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

package messagelayer

import (
	"fmt"
	"testing"

	"github.com/kubeedge/beehive/pkg/core/model"
	"github.com/kubeedge/kubeedge/common/constants"
)

const (
	NodeID       = "NodeA"
	ResourceID   = "ResA"
	Namespace    = "default"
	ResourceType = "pod"
)

func TestBuildResourceForDevice(t *testing.T) {
	type args struct {
		nodeID       string
		resourceType string
		resourceID   string
	}
	tests := []struct {
		name         string
		args         args
		wantResource string
		wantErr      error
	}{
		{
			name: "TestBuildResourceForDevice(): Case 1: Test with nodeID, resourceType and resourceID",
			args: args{
				nodeID:       "nid",
				resourceType: constants.ResourceTypePersistentVolume,
				resourceID:   "rid",
			},
			wantResource: fmt.Sprintf("%s%s%s%s%s%s%s", ResourceNode, constants.ResourceSep, "nid", constants.ResourceSep, constants.ResourceTypePersistentVolume, constants.ResourceSep, "rid"),
			wantErr:      nil,
		},
		{
			name: "TestBuildResourceForDevice(): Case 2: Test without nodeID",
			args: args{
				nodeID:       "",
				resourceType: "",
				resourceID:   "",
			},
			wantResource: "",
			wantErr:      fmt.Errorf("required parameter are not set (node id, namespace or resource type)"),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotResource, err := BuildResourceForDevice(tt.args.nodeID, tt.args.resourceType, tt.args.resourceID)
			if err != nil && err.Error() != tt.wantErr.Error() {
				t.Errorf("BuildResource() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if gotResource != tt.wantResource {
				t.Errorf("BuildResourceForDevice() gotResource = %v, want %v", gotResource, tt.wantResource)
			}
		})
	}
}

func TestGetDeviceID(t *testing.T) {
	type args struct {
		resource string
	}
	tests := []struct {
		name    string
		args    args
		want    string
		wantErr error
	}{
		{
			name: "TestGetDeviceID(): Case 1: success",
			args: args{
				resource: fmt.Sprintf("node/%s/%s/%s", "nid", ResourceDevice, "did"),
			},
			want:    "did",
			wantErr: nil,
		},
		{
			name: "TestGetDeviceID(): Case 2: length less then 4",
			args: args{
				resource: fmt.Sprintf("node/%s/%s", "nid", ResourceDevice),
			},
			want:    "",
			wantErr: fmt.Errorf("failed to get device id"),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := GetDeviceID(tt.args.resource)
			if err != nil && err.Error() != tt.wantErr.Error() {
				t.Errorf("GetDeviceID() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("GetDeviceID() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGetNodeID(t *testing.T) {
	type args struct {
		msg model.Message
	}
	tests := []struct {
		name    string
		args    args
		want    string
		wantErr error
	}{
		{
			"TestGetNodeID(): Case 1: success",
			args{
				msg: model.Message{
					Router: model.MessageRoute{
						Resource: fmt.Sprintf("node/%s/%s/%s/%s", "nid", "default", constants.ResourceTypeEndpoints, "rid"),
					},
				},
			},
			"nid",
			nil,
		},
		{
			"TestGetNodeID(): Case 2: no nodeID",
			args{msg: model.Message{}},
			"",
			fmt.Errorf("node id not found"),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := GetNodeID(tt.args.msg)
			if err != nil && err.Error() != tt.wantErr.Error() {
				t.Errorf("GetNodeID() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("GetNodeID() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGetResourceTypeForDevice(t *testing.T) {
	type args struct {
		resource string
	}
	tests := []struct {
		name    string
		args    args
		want    string
		wantErr error
	}{
		{
			"GetResourceTypeForDevice ResourceTypeTwinEdgeUpdated: success",
			args{
				resource: fmt.Sprintf("node/%s/%s", "nid", ResourceTypeTwinEdgeUpdated),
			},
			ResourceTypeTwinEdgeUpdated,
			nil,
		},
		{
			"GetResourceTypeForDevice() ResourceTypeMembershipDetail: success",
			args{
				resource: ResourceTypeMembershipDetail,
			},
			ResourceTypeMembershipDetail,
			nil,
		},
		{
			"GetResourceTypeForDevice() Case 2: no resourceType",
			args{
				resource: "",
			},
			"",
			fmt.Errorf("unknown resource, found: %s", ""),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := GetResourceTypeForDevice(tt.args.resource)
			if err != nil && err.Error() != tt.wantErr.Error() {
				t.Errorf("GetResourceType() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("GetResourceType() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestBuildResource(t *testing.T) {
	type args struct {
		nodeID       string
		namespace    string
		resourceType string
		resourceID   string
	}
	tests := []struct {
		name         string
		args         args
		wantResource string
		wantErr      error
	}{
		{
			"TestBuildResource(): Case 1: no node ID, no namespace.",
			args{
				nodeID:       "",
				namespace:    "",
				resourceType: ResourceType,
				resourceID:   ResourceID,
			},
			"",
			fmt.Errorf("required parameter are not set (node id, namespace or resource type)"),
		},
		{
			"TestBuildResource(): Case 2: no resourceID.",
			args{
				nodeID:       NodeID,
				namespace:    Namespace,
				resourceType: ResourceType,
				resourceID:   "",
			},
			"node/" + NodeID + "/" + Namespace + "/" + ResourceType,
			nil,
		},
		{
			"TestBuildResource(): Case 1: with resourceID.",
			args{
				nodeID:       NodeID,
				namespace:    Namespace,
				resourceType: ResourceType,
				resourceID:   ResourceID,
			},
			"node/" + NodeID + "/" + Namespace + "/" + ResourceType + "/" + ResourceID,
			nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotResource, err := BuildResource(tt.args.nodeID, tt.args.namespace, tt.args.resourceType, tt.args.resourceID)
			if err != nil && err.Error() != tt.wantErr.Error() {
				t.Errorf("BuildResource() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if gotResource != tt.wantResource {
				t.Errorf("BuildResource() = %v, want %v", gotResource, tt.wantResource)
			}
		})
	}
}

func TestBuildResourceForRouter(t *testing.T) {
	type args struct {
		resourceType string
		resourceID   string
	}
	tests := []struct {
		name         string
		args         args
		wantResource string
		wantErr      error
	}{
		{
			"TestBuildResourceForRouter(): Case 1: no resourceType.",
			args{
				resourceType: "",
				resourceID:   ResourceID,
			},
			"",
			fmt.Errorf("required parameter are not set (resourceID or resource type)"),
		},
		{
			"TestBuildResourceForRouter(): Case 2: no resourceID.",
			args{
				resourceType: ResourceType,
				resourceID:   "",
			},
			"",
			fmt.Errorf("required parameter are not set (resourceID or resource type)"),
		},
		{
			"TestBuildResourceForRouter(): Case 3: success.",
			args{
				resourceType: ResourceType,
				resourceID:   ResourceID,
			},
			ResourceType + "/" + ResourceID,
			nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotResource, err := BuildResourceForRouter(tt.args.resourceType, tt.args.resourceID)
			if err != nil && err.Error() != tt.wantErr.Error() {
				t.Errorf("BuildResourceForRouter() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if gotResource != tt.wantResource {
				t.Errorf("BuildResourceForRouter() = %v, want %v", gotResource, tt.wantResource)
			}
		})
	}
}

func TestGetNamespace(t *testing.T) {
	type args struct {
		msg model.Message
	}

	tests := []struct {
		name    string
		args    args
		want    string
		wantErr error
	}{
		{
			"TestGetNodeID() Case 1: has namespace",
			args{
				msg: model.Message{
					Router: model.MessageRoute{
						Resource: fmt.Sprintf("node/%s/%s/%s/%s", NodeID, Namespace, ResourceType, ResourceID),
					},
				},
			},
			Namespace,
			nil,
		},
		{
			"TestGetNodeID() Case 2: no namespace",
			args{
				msg: model.Message{},
			},
			"",
			fmt.Errorf("namespace not found"),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := GetNamespace(tt.args.msg)
			if err != nil && err.Error() != tt.wantErr.Error() {
				t.Errorf("GetNamespace() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("GetNamespace() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGetResourceType(t *testing.T) {
	type args struct {
		msg model.Message
	}
	tests := []struct {
		name    string
		args    args
		want    string
		wantErr error
	}{
		{
			"TestGetNodeID() Case 1: has resourceType",
			args{
				msg: model.Message{
					Router: model.MessageRoute{
						Resource: fmt.Sprintf("node/%s/%s/%s/%s", NodeID, Namespace, ResourceType, ResourceID),
					},
				},
			},
			ResourceType,
			nil,
		},
		{
			"TestGetNodeID() Case 2: no resourceType",
			args{
				msg: model.Message{},
			},
			"",
			fmt.Errorf("resource type not found"),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := GetResourceType(tt.args.msg)
			if err != nil && err.Error() != tt.wantErr.Error() {
				t.Errorf("GetResourceType() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("GetResourceType() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGetResourceName(t *testing.T) {
	type args struct {
		msg model.Message
	}

	tests := []struct {
		name    string
		args    args
		want    string
		wantErr error
	}{
		{
			"TestGetNodeID() Case 1: has resourceName",
			args{
				msg: model.Message{
					Router: model.MessageRoute{
						Resource: fmt.Sprintf("node/%s/%s/%s/%s", NodeID, Namespace, ResourceType, ResourceID),
					},
				},
			},
			ResourceID,
			nil,
		},
		{
			"TestGetNodeID() Case 3: no resourceName",
			args{
				msg: model.Message{},
			},
			"",
			fmt.Errorf("resource name not found"),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := GetResourceName(tt.args.msg)
			if err != nil && err.Error() != tt.wantErr.Error() {
				t.Errorf("GetResourceName() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("GetResourceName() = %v, want %v", got, tt.want)
			}
		})
	}
}
