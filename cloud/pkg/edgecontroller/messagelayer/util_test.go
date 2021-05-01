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
	"github.com/kubeedge/kubeedge/cloud/pkg/edgecontroller/config"
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
		isEdgeSite   bool
	}
	tests := []struct {
		name         string
		args         args
		wantResource string
		wantErr      error
	}{
		{
			"TestBuildResource(): Case 1: not edgesite, no node ID, no namespace.",
			args{
				nodeID:       "",
				namespace:    "",
				resourceType: ResourceType,
				resourceID:   ResourceID,
				isEdgeSite:   false,
			},
			"",
			fmt.Errorf("required parameter are not set (node id, namespace or resource type)"),
		},
		{
			"TestBuildResource(): Case 2: is edgesite, no node ID, no namespace ",
			args{
				nodeID:       "",
				namespace:    "",
				resourceType: ResourceType,
				resourceID:   ResourceID,
				isEdgeSite:   true,
			},
			"",
			fmt.Errorf("required parameter are not set (namespace or resource type)"),
		},
		{
			"TestBuildResource(): Case 3: is edgesite, has nodeID, namespace",
			args{
				nodeID:       NodeID,
				namespace:    Namespace,
				resourceType: ResourceType,
				resourceID:   ResourceID,
				isEdgeSite:   true,
			},
			fmt.Sprintf("%s/%s/%s", Namespace, ResourceType, ResourceID),
			nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config.Config.EdgeSiteEnable = tt.args.isEdgeSite
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
		msg        model.Message
		isEdgeSite bool
	}

	tests := []struct {
		name    string
		args    args
		want    string
		wantErr error
	}{
		{
			"TestGetNodeID() Case 1: is not edgesite, has namespace",
			args{
				msg: model.Message{
					Router: model.MessageRoute{
						Resource: fmt.Sprintf("node/%s/%s/%s/%s", NodeID, Namespace, ResourceType, ResourceID),
					},
				},
				isEdgeSite: false,
			},
			Namespace,
			nil,
		},
		{
			"TestGetNodeID() Case 2: is edgesite, has namespace",
			args{
				msg: model.Message{
					Router: model.MessageRoute{
						Resource: fmt.Sprintf("%s/%s/%s", Namespace, ResourceType, ResourceID),
					},
				},
				isEdgeSite: true,
			},
			Namespace,
			nil,
		},
		{
			"TestGetNodeID() Case 3: is edgesite, no namespace",
			args{
				msg:        model.Message{},
				isEdgeSite: true,
			},
			"",
			fmt.Errorf("namespace not found"),
		},
		{
			"TestGetNodeID() Case 4: not edgesite, no namespace",
			args{
				msg:        model.Message{},
				isEdgeSite: false,
			},
			"",
			fmt.Errorf("namespace not found"),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config.Config.EdgeSiteEnable = tt.args.isEdgeSite
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
		msg        model.Message
		isEdgeSite bool
	}
	tests := []struct {
		name    string
		args    args
		want    string
		wantErr error
	}{
		{
			"TestGetNodeID() Case 1: is not edgesite, has resourceType",
			args{
				msg: model.Message{
					Router: model.MessageRoute{
						Resource: fmt.Sprintf("node/%s/%s/%s/%s", NodeID, Namespace, ResourceType, ResourceID),
					},
				},
				isEdgeSite: false,
			},
			ResourceType,
			nil,
		},
		{
			"TestGetNodeID() Case 2: is edgesite, has resourceType",
			args{
				msg: model.Message{
					Router: model.MessageRoute{
						Resource: fmt.Sprintf("%s/%s/%s", Namespace, ResourceType, ResourceID),
					},
				},
				isEdgeSite: true,
			},
			ResourceType,
			nil,
		},
		{
			"TestGetNodeID() Case 3: no resourceType",
			args{
				msg:        model.Message{},
				isEdgeSite: true,
			},
			"",
			fmt.Errorf("resource type not found"),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config.Config.EdgeSiteEnable = tt.args.isEdgeSite
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
		msg        model.Message
		isEdgeSite bool
	}

	tests := []struct {
		name    string
		args    args
		want    string
		wantErr error
	}{
		{
			"TestGetNodeID() Case 1: is not edgesite, has resourceName",
			args{
				msg: model.Message{
					Router: model.MessageRoute{
						Resource: fmt.Sprintf("node/%s/%s/%s/%s", NodeID, Namespace, ResourceType, ResourceID),
					},
				},
				isEdgeSite: false,
			},
			ResourceID,
			nil,
		},
		{
			"TestGetNodeID() Case 2: is edgesite, has resourceName",
			args{
				msg: model.Message{
					Router: model.MessageRoute{
						Resource: fmt.Sprintf("%s/%s/%s", Namespace, ResourceType, ResourceID),
					},
				},
				isEdgeSite: true,
			},
			ResourceID,
			nil,
		},
		{
			"TestGetNodeID() Case 3: no resourceName",
			args{
				msg:        model.Message{},
				isEdgeSite: true,
			},
			"",
			fmt.Errorf("resource name not found"),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config.Config.EdgeSiteEnable = tt.args.isEdgeSite
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
