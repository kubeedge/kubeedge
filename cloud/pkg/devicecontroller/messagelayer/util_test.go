package messagelayer

import (
	"fmt"
	"testing"

	"github.com/kubeedge/beehive/pkg/core/model"
	deviceconstants "github.com/kubeedge/kubeedge/cloud/pkg/devicecontroller/constants"
	"github.com/kubeedge/kubeedge/common/constants"
)

func TestBuildResource(t *testing.T) {
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
			name: "TestBuildResource(): Case 1: Test with nodeID, resourceType and resourceID",
			args: args{
				nodeID:       "nid",
				resourceType: constants.ResourceTypePersistentVolume,
				resourceID:   "rid",
			},
			wantResource: fmt.Sprintf("%s%s%s%s%s%s%s", deviceconstants.ResourceNode, constants.ResourceSep, "nid", constants.ResourceSep, constants.ResourceTypePersistentVolume, constants.ResourceSep, "rid"),
			wantErr:      nil,
		},
		{
			name: "TestBuildResource(): Case 2: Test without nodeID",
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
			gotResource, err := BuildResource(tt.args.nodeID, tt.args.resourceType, tt.args.resourceID)
			if err != nil && err.Error() != tt.wantErr.Error() {
				t.Errorf("BuildResource() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if gotResource != tt.wantResource {
				t.Errorf("BuildResource() gotResource = %v, want %v", gotResource, tt.wantResource)
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
				resource: fmt.Sprintf("node/%s/%s/%s", "nid", deviceconstants.ResourceDevice, "did"),
			},
			want:    "did",
			wantErr: nil,
		},
		{
			name: "TestGetDeviceID(): Case 2: length less then 4",
			args: args{
				resource: fmt.Sprintf("node/%s/%s", "nid", deviceconstants.ResourceDevice),
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

func TestGetResourceType(t *testing.T) {
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
			"TestGetNodeID() Case 1: success",
			args{
				resource: fmt.Sprintf("node/%s/%s", "nid", deviceconstants.ResourceTypeTwinEdgeUpdated),
			},
			deviceconstants.ResourceTypeTwinEdgeUpdated,
			nil,
		},
		{
			"TestGetNodeID() Case 2: no resourceType",
			args{
				resource: "",
			},
			"",
			fmt.Errorf("unknown resource"),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := GetResourceType(tt.args.resource)
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
