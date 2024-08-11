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

package csidriver

import (
	"testing"

	"github.com/container-storage-interface/spec/lib/go/csi"
	"github.com/stretchr/testify/assert"
	"golang.org/x/net/context"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestNewControllerServer(t *testing.T) {
	assert := assert.New(t)

	nodeID := "test-node"
	kubeEdgeEndpoint := "http://localhost:8080/test"

	cs := newControllerServer(nodeID, kubeEdgeEndpoint)
	assert.NotNil(cs)

	assert.Equal(nodeID, cs.nodeID)
	assert.Equal(kubeEdgeEndpoint, cs.kubeEdgeEndpoint)

	expectedCaps := getControllerServiceCapabilities(
		[]csi.ControllerServiceCapability_RPC_Type{
			csi.ControllerServiceCapability_RPC_CREATE_DELETE_VOLUME,
			csi.ControllerServiceCapability_RPC_PUBLISH_UNPUBLISH_VOLUME,
		})
	assert.Equal(expectedCaps, cs.caps)

	assert.Equal(csi.ControllerServiceCapability_RPC_CREATE_DELETE_VOLUME,
		cs.caps[0].GetRpc().GetType())
	assert.Equal(csi.ControllerServiceCapability_RPC_PUBLISH_UNPUBLISH_VOLUME,
		cs.caps[1].GetRpc().GetType())
}

func TestValidateVolumeCapabilities(t *testing.T) {
	assert := assert.New(t)

	cs := &controllerServer{
		nodeID:           "test-node",
		kubeEdgeEndpoint: "http://localhost:8080/test",
	}

	// Test case 1: Invalid request (missing volume ID)
	invalidReq := &csi.ValidateVolumeCapabilitiesRequest{
		VolumeCapabilities: []*csi.VolumeCapability{
			{
				AccessType: &csi.VolumeCapability_Mount{
					Mount: &csi.VolumeCapability_MountVolume{},
				},
				AccessMode: &csi.VolumeCapability_AccessMode{
					Mode: csi.VolumeCapability_AccessMode_SINGLE_NODE_WRITER,
				},
			},
		},
	}

	result, err := cs.ValidateVolumeCapabilities(context.Background(), invalidReq)
	assert.Error(err)
	assert.Nil(result)
	assert.Equal(codes.InvalidArgument, status.Code(err))
	assert.Contains(err.Error(), "Volume ID cannot be empty")

	// Test case 2: Invalid request (missing volume capabilities)
	invalidReq2 := &csi.ValidateVolumeCapabilitiesRequest{
		VolumeId: "test-volume-id",
	}

	result, err = cs.ValidateVolumeCapabilities(context.Background(), invalidReq2)
	assert.Error(err)
	assert.Nil(result)
	assert.Equal(codes.InvalidArgument, status.Code(err))
	assert.Contains(err.Error(), "test-volume-id")

	// Test case 3: Valid request
	validReq := &csi.ValidateVolumeCapabilitiesRequest{
		VolumeId: "test-volume-id",
		VolumeCapabilities: []*csi.VolumeCapability{
			{
				AccessType: &csi.VolumeCapability_Mount{
					Mount: &csi.VolumeCapability_MountVolume{},
				},
				AccessMode: &csi.VolumeCapability_AccessMode{
					Mode: csi.VolumeCapability_AccessMode_SINGLE_NODE_WRITER,
				},
			},
		},
	}

	result, err = cs.ValidateVolumeCapabilities(context.Background(), validReq)
	assert.NoError(err)
	assert.NotNil(result)
	assert.NotNil(result.Confirmed)
	assert.NotEmpty(result.Confirmed.VolumeCapabilities)
	assert.Equal(csi.VolumeCapability_AccessMode_SINGLE_NODE_WRITER,
		result.Confirmed.VolumeCapabilities[0].AccessMode.Mode)
}

func TestControllerGetCapabilities(t *testing.T) {
	assert := assert.New(t)

	cs := &controllerServer{
		nodeID:           "test-node",
		kubeEdgeEndpoint: "http://localhost:8080/test",
		caps: getControllerServiceCapabilities(
			[]csi.ControllerServiceCapability_RPC_Type{
				csi.ControllerServiceCapability_RPC_CREATE_DELETE_VOLUME,
				csi.ControllerServiceCapability_RPC_PUBLISH_UNPUBLISH_VOLUME,
			},
		),
	}

	req := &csi.ControllerGetCapabilitiesRequest{}
	resp, err := cs.ControllerGetCapabilities(context.Background(), req)
	assert.NoError(err)
	assert.NotNil(resp)

	expectedCaps := []csi.ControllerServiceCapability_RPC_Type{
		csi.ControllerServiceCapability_RPC_CREATE_DELETE_VOLUME,
		csi.ControllerServiceCapability_RPC_PUBLISH_UNPUBLISH_VOLUME,
	}

	for i, cap := range resp.Capabilities {
		assert.Equal(expectedCaps[i], cap.GetRpc().Type,
			"Capability %d should be %v", i, expectedCaps[i])
	}
}

func TestGetControllerServiceCapabilities(t *testing.T) {
	assert := assert.New(t)

	// Test case 1: Empty capability list
	emptyCaps := getControllerServiceCapabilities([]csi.ControllerServiceCapability_RPC_Type{})
	assert.Empty(emptyCaps)

	// Test case 2: One capability
	singleCapType := csi.ControllerServiceCapability_RPC_CREATE_DELETE_VOLUME
	singleCap := getControllerServiceCapabilities([]csi.ControllerServiceCapability_RPC_Type{singleCapType})

	assert.Equal(singleCapType, singleCap[0].GetRpc().Type)

	// Test case 3: Multiple capabilities
	multiCapTypes := []csi.ControllerServiceCapability_RPC_Type{
		csi.ControllerServiceCapability_RPC_CREATE_DELETE_VOLUME,
		csi.ControllerServiceCapability_RPC_PUBLISH_UNPUBLISH_VOLUME,
		csi.ControllerServiceCapability_RPC_LIST_VOLUMES,
	}
	multiCaps := getControllerServiceCapabilities(multiCapTypes)

	assert.Len(multiCaps, 3)
	for i, capType := range multiCapTypes {
		assert.Equal(capType, multiCaps[i].GetRpc().Type,
			"Capability %d should be %v", i, capType)
	}
}
