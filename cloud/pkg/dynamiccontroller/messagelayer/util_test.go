/*
Copyright 2021 The KubeEdge Authors.

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
)

const (
	NodeID       = "NodeA"
	ResourceID   = "ResA"
	Namespace    = "default"
	ResourceType = "pod"
)

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
			"TestBuildResource(): Case 3: with resourceID.",
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
			"TestGetNodeID() Case 1: has node ID",
			args{
				msg: model.Message{
					Router: model.MessageRoute{
						Resource: fmt.Sprintf("node/%s/%s/%s/%s", NodeID, Namespace, ResourceType, ResourceID),
					},
				},
			},
			NodeID,
			nil,
		},
		{
			"TestGetNodeID() Case 2: no node ID",
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
				t.Errorf("GetNodeID() = %v, want %v", got, tt.want)
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
