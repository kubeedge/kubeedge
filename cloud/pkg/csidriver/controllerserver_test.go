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
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"testing"

	"github.com/container-storage-interface/spec/lib/go/csi"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/kubeedge/beehive/pkg/core/model"
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
func TestCreateVolume(t *testing.T) {
	assert := assert.New(t)

	cs := &controllerServer{
		nodeID:           "test-node",
		kubeEdgeEndpoint: "http://localhost:8080/test",
	}

	// Test case 1: Invalid request (missing name)
	invalidReq := &csi.CreateVolumeRequest{
		VolumeCapabilities: []*csi.VolumeCapability{
			{
				AccessType: &csi.VolumeCapability_Mount{
					Mount: &csi.VolumeCapability_MountVolume{},
				},
			},
		},
	}

	result, err := cs.CreateVolume(context.Background(), invalidReq)
	assert.Error(err)
	assert.Nil(result)
	assert.Equal(codes.InvalidArgument, status.Code(err))
	assert.Contains(err.Error(), "Name missing in request")

	// Test case 2: Invalid request (missing volume capabilities)
	invalidReq2 := &csi.CreateVolumeRequest{
		Name: "test-volume",
	}

	result, err = cs.CreateVolume(context.Background(), invalidReq2)
	assert.Error(err)
	assert.Nil(result)
	assert.Equal(codes.InvalidArgument, status.Code(err))
	assert.Contains(err.Error(), "Volume Capabilities missing in request")
}

func TestValidateVolumeCapabilitiesEdgeCases(t *testing.T) {
	assert := assert.New(t)

	cs := &controllerServer{
		nodeID:           "test-node",
		kubeEdgeEndpoint: "http://localhost:8080/test",
	}

	// Test case: Invalid request (undefined mount and block)
	invalidReq := &csi.ValidateVolumeCapabilitiesRequest{
		VolumeId: "test-volume",
		VolumeCapabilities: []*csi.VolumeCapability{
			{
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
	assert.Contains(err.Error(), "cannot have both mount and block access type be undefined")
}

func TestUnimplementedMethods(t *testing.T) {
	assert := assert.New(t)

	cs := &controllerServer{}
	ctx := context.Background()

	// Test GetCapacity
	capResp, err := cs.GetCapacity(ctx, &csi.GetCapacityRequest{})
	assert.Error(err)
	assert.Nil(capResp)
	assert.Equal(codes.Unimplemented, status.Code(err))

	// Test ListVolumes
	listResp, err := cs.ListVolumes(ctx, &csi.ListVolumesRequest{})
	assert.Error(err)
	assert.Nil(listResp)
	assert.Equal(codes.Unimplemented, status.Code(err))

	// Test ControllerExpandVolume
	expandResp, err := cs.ControllerExpandVolume(ctx, &csi.ControllerExpandVolumeRequest{})
	assert.Error(err)
	assert.Nil(expandResp)
	assert.Equal(codes.Unimplemented, status.Code(err))

	// Test CreateSnapshot
	snapResp, err := cs.CreateSnapshot(ctx, &csi.CreateSnapshotRequest{})
	assert.Error(err)
	assert.Nil(snapResp)
	assert.Equal(codes.Unimplemented, status.Code(err))

	// Test DeleteSnapshot
	delSnapResp, err := cs.DeleteSnapshot(ctx, &csi.DeleteSnapshotRequest{})
	assert.Error(err)
	assert.Nil(delSnapResp)
	assert.Equal(codes.Unimplemented, status.Code(err))

	// Test ListSnapshots
	listSnapResp, err := cs.ListSnapshots(ctx, &csi.ListSnapshotsRequest{})
	assert.Error(err)
	assert.Nil(listSnapResp)
	assert.Equal(codes.Unimplemented, status.Code(err))

	// Test ControllerGetVolume
	getVolResp, err := cs.ControllerGetVolume(ctx, &csi.ControllerGetVolumeRequest{})
	assert.Error(err)
	assert.Nil(getVolResp)
	assert.Equal(codes.Unimplemented, status.Code(err))
}

func mockUnixSocketServer(t *testing.T) (string, func()) {
	dir, err := os.MkdirTemp("", "csi-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}

	socketPath := filepath.Join(dir, "test.sock")
	listener, err := net.Listen("unix", socketPath)
	if err != nil {
		t.Fatalf("Failed to create Unix domain socket: %v", err)
	}

	go func() {
		for {
			conn, err := listener.Accept()
			if err != nil {
				return
			}
			go handleConnection(conn)
		}
	}()

	cleanup := func() {
		listener.Close()
		os.RemoveAll(dir)
	}

	return "unix://" + socketPath, cleanup
}

func handleConnection(conn net.Conn) {
	defer conn.Close()

	buf := make([]byte, DefaultBufferSize)
	_, err := conn.Read(buf)
	if err != nil {
		return
	}

	resp := model.Message{
		Content: base64.StdEncoding.EncodeToString([]byte(`{"volume":{"volume_id":"test-volume-id"}}`)),
	}

	respBytes, _ := json.Marshal(resp)
	if _, err := conn.Write(respBytes); err != nil {
		return
	}
}

func TestCreateVolumeValidation(t *testing.T) {
	assert := assert.New(t)

	socketPath, cleanup := mockUnixSocketServer(t)
	defer cleanup()

	cs := &controllerServer{
		nodeID:           "test-node",
		kubeEdgeEndpoint: socketPath,
	}

	testCases := []struct {
		name          string
		req           *csi.CreateVolumeRequest
		expectedError codes.Code
		errorContains string
	}{
		{
			name: "Empty volume name",
			req: &csi.CreateVolumeRequest{
				VolumeCapabilities: []*csi.VolumeCapability{
					{
						AccessType: &csi.VolumeCapability_Mount{
							Mount: &csi.VolumeCapability_MountVolume{},
						},
					},
				},
			},
			expectedError: codes.InvalidArgument,
			errorContains: "Name missing in request",
		},
		{
			name: "Missing volume capabilities",
			req: &csi.CreateVolumeRequest{
				Name: "test-volume",
			},
			expectedError: codes.InvalidArgument,
			errorContains: "Volume Capabilities missing in request",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			resp, err := cs.CreateVolume(context.Background(), tc.req)
			assert.Error(err)
			assert.Nil(resp)
			assert.Equal(tc.expectedError, status.Code(err))
			assert.Contains(err.Error(), tc.errorContains)
		})
	}
}

func TestControllerPublishVolumeValidation(t *testing.T) {
	assert := assert.New(t)

	socketPath, cleanup := mockUnixSocketServer(t)
	defer cleanup()

	cs := &controllerServer{
		nodeID:           "test-node",
		kubeEdgeEndpoint: socketPath,
	}

	testCases := []struct {
		name          string
		req           *csi.ControllerPublishVolumeRequest
		expectedError codes.Code
		errorContains string
	}{
		{
			name:          "Missing volume ID",
			req:           &csi.ControllerPublishVolumeRequest{},
			expectedError: codes.InvalidArgument,
			errorContains: "Volume ID must be provided",
		},
		{
			name: "Missing node ID",
			req: &csi.ControllerPublishVolumeRequest{
				VolumeId: "test-volume",
			},
			expectedError: codes.InvalidArgument,
			errorContains: "Instance ID must be provided",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			resp, err := cs.ControllerPublishVolume(context.Background(), tc.req)
			assert.Error(err)
			assert.Nil(resp)
			assert.Equal(tc.expectedError, status.Code(err))
			assert.Contains(err.Error(), tc.errorContains)
		})
	}
}

func TestSuccessfulOperations(t *testing.T) {
	assert := assert.New(t)

	socketPath, cleanup := mockUnixSocketServer(t)
	defer cleanup()

	cs := &controllerServer{
		nodeID:           "test-node",
		kubeEdgeEndpoint: socketPath,
		caps: getControllerServiceCapabilities([]csi.ControllerServiceCapability_RPC_Type{
			csi.ControllerServiceCapability_RPC_CREATE_DELETE_VOLUME,
			csi.ControllerServiceCapability_RPC_PUBLISH_UNPUBLISH_VOLUME,
		}),
	}

	createReq := &csi.CreateVolumeRequest{
		Name: "test-volume",
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
		CapacityRange: &csi.CapacityRange{
			RequiredBytes: 1024 * 1024 * 1024,
		},
	}

	createResp, err := cs.CreateVolume(context.Background(), createReq)
	assert.NoError(err)
	assert.NotNil(createResp)
	assert.NotEmpty(createResp.Volume.VolumeId)

	capsReq := &csi.ControllerGetCapabilitiesRequest{}
	capsResp, err := cs.ControllerGetCapabilities(context.Background(), capsReq)
	assert.NoError(err)
	assert.NotNil(capsResp)
	assert.Len(capsResp.Capabilities, 2)
}

func TestValidateVolumeCapabilitiesValidation(t *testing.T) {
	assert := assert.New(t)

	cs := &controllerServer{
		nodeID:           "test-node",
		kubeEdgeEndpoint: "unix:///tmp/test.sock",
	}

	testCases := []struct {
		name          string
		req           *csi.ValidateVolumeCapabilitiesRequest
		expectedError codes.Code
		errorContains string
	}{
		{
			name: "Empty volume ID",
			req: &csi.ValidateVolumeCapabilitiesRequest{
				VolumeCapabilities: []*csi.VolumeCapability{
					{
						AccessType: &csi.VolumeCapability_Mount{},
					},
				},
			},
			expectedError: codes.InvalidArgument,
			errorContains: "Volume ID cannot be empty",
		},
		{
			name: "Empty capabilities",
			req: &csi.ValidateVolumeCapabilitiesRequest{
				VolumeId: "test-volume",
			},
			expectedError: codes.InvalidArgument,
			errorContains: "test-volume",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			resp, err := cs.ValidateVolumeCapabilities(context.Background(), tc.req)
			assert.Error(err)
			assert.Nil(resp)
			assert.Equal(tc.expectedError, status.Code(err))
			assert.Contains(err.Error(), tc.errorContains)
		})
	}
}

type mockSocketServer struct {
	listener net.Listener
	t        *testing.T
	response *model.Message
}

func newMockServer(t *testing.T) (*mockSocketServer, string, func()) {
	dir, err := os.MkdirTemp("", "csi-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}

	socketPath := filepath.Join(dir, "test.sock")
	listener, err := net.Listen("unix", socketPath)
	if err != nil {
		t.Fatalf("Failed to create Unix domain socket: %v", err)
	}

	server := &mockSocketServer{
		listener: listener,
		t:        t,
		response: &model.Message{
			Content: base64.StdEncoding.EncodeToString([]byte(`{}`)),
		},
	}

	go server.serve()

	cleanup := func() {
		listener.Close()
		os.RemoveAll(dir)
	}

	return server, "unix://" + socketPath, cleanup
}

func (m *mockSocketServer) serve() {
	for {
		conn, err := m.listener.Accept()
		if err != nil {
			return
		}
		go m.handleConnection(conn)
	}
}

func (m *mockSocketServer) handleConnection(conn net.Conn) {
	defer conn.Close()

	buf := make([]byte, DefaultBufferSize)
	_, err := conn.Read(buf)
	if err != nil {
		m.t.Errorf("Failed to read from connection: %v", err)
		return
	}

	respBytes, err := json.Marshal(m.response)
	if err != nil {
		m.t.Errorf("Failed to marshal response: %v", err)
		return
	}

	_, err = conn.Write(respBytes)
	if err != nil {
		m.t.Errorf("Failed to write response: %v", err)
		return
	}
}

func (m *mockSocketServer) setResponse(resp interface{}) {
	respBytes, err := json.Marshal(resp)
	if err != nil {
		m.t.Fatalf("Failed to marshal mock response: %v", err)
	}
	m.response.Content = base64.StdEncoding.EncodeToString(respBytes)
}

func TestDeleteVolume(t *testing.T) {
	assert := assert.New(t)

	mockServer, socketPath, cleanup := newMockServer(t)
	defer cleanup()

	cs := &controllerServer{
		nodeID:           "test-node",
		kubeEdgeEndpoint: socketPath,
	}

	invalidReq := &csi.DeleteVolumeRequest{}
	resp, err := cs.DeleteVolume(context.Background(), invalidReq)
	assert.Error(err)
	assert.Nil(resp)
	assert.Equal(codes.InvalidArgument, status.Code(err))
	assert.Contains(err.Error(), "Volume ID missing in request")

	mockServer.setResponse(&csi.DeleteVolumeResponse{})
	validReq := &csi.DeleteVolumeRequest{
		VolumeId: "test-volume",
	}
	resp, err = cs.DeleteVolume(context.Background(), validReq)
	assert.NoError(err)
	assert.NotNil(resp)
}

func TestControllerPublishVolume(t *testing.T) {
	assert := assert.New(t)

	mockServer, socketPath, cleanup := newMockServer(t)
	defer cleanup()

	cs := &controllerServer{
		nodeID:           "test-node",
		kubeEdgeEndpoint: socketPath,
	}

	testCases := []struct {
		name          string
		req           *csi.ControllerPublishVolumeRequest
		expectedError codes.Code
		errorContains string
		mockResponse  interface{}
	}{
		{
			name:          "Missing volume ID",
			req:           &csi.ControllerPublishVolumeRequest{NodeId: "test-node"},
			expectedError: codes.InvalidArgument,
			errorContains: "Volume ID must be provided",
		},
		{
			name:          "Missing node ID",
			req:           &csi.ControllerPublishVolumeRequest{VolumeId: "test-volume"},
			expectedError: codes.InvalidArgument,
			errorContains: "Instance ID must be provided",
		},
		{
			name: "Successful publish",
			req: &csi.ControllerPublishVolumeRequest{
				VolumeId: "test-volume",
				NodeId:   "test-node",
				VolumeCapability: &csi.VolumeCapability{
					AccessType: &csi.VolumeCapability_Mount{
						Mount: &csi.VolumeCapability_MountVolume{},
					},
				},
			},
			mockResponse: &csi.ControllerPublishVolumeResponse{
				PublishContext: map[string]string{"devicePath": "/dev/test"},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.mockResponse != nil {
				mockServer.setResponse(tc.mockResponse)
			}

			resp, err := cs.ControllerPublishVolume(context.Background(), tc.req)

			if tc.expectedError != codes.OK {
				assert.Error(err)
				assert.Nil(resp)
				assert.Equal(tc.expectedError, status.Code(err))
				assert.Contains(err.Error(), tc.errorContains)
			} else {
				assert.NoError(err)
				assert.NotNil(resp)
				assert.NotNil(resp.PublishContext)
			}
		})
	}
}

func TestControllerUnpublishVolume(t *testing.T) {
	assert := assert.New(t)

	mockServer, socketPath, cleanup := newMockServer(t)
	defer cleanup()

	cs := &controllerServer{
		nodeID:           "test-node",
		kubeEdgeEndpoint: socketPath,
	}

	testCases := []struct {
		name          string
		req           *csi.ControllerUnpublishVolumeRequest
		expectedError codes.Code
		errorContains string
		mockResponse  interface{}
	}{
		{
			name:          "Missing volume ID",
			req:           &csi.ControllerUnpublishVolumeRequest{NodeId: "test-node"},
			expectedError: codes.InvalidArgument,
			errorContains: "Volume ID must be provided",
		},
		{
			name:          "Missing node ID",
			req:           &csi.ControllerUnpublishVolumeRequest{VolumeId: "test-volume"},
			expectedError: codes.InvalidArgument,
			errorContains: "Instance ID must be provided",
		},
		{
			name: "Successful unpublish",
			req: &csi.ControllerUnpublishVolumeRequest{
				VolumeId: "test-volume",
				NodeId:   "test-node",
			},
			mockResponse: &csi.ControllerUnpublishVolumeResponse{},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.mockResponse != nil {
				mockServer.setResponse(tc.mockResponse)
			}

			resp, err := cs.ControllerUnpublishVolume(context.Background(), tc.req)

			if tc.expectedError != codes.OK {
				assert.Error(err)
				assert.Nil(resp)
				assert.Equal(tc.expectedError, status.Code(err))
				assert.Contains(err.Error(), tc.errorContains)
			} else {
				assert.NoError(err)
				assert.NotNil(resp)
			}
		})
	}
}

func TestErrorResponses(t *testing.T) {
	assert := assert.New(t)

	mockServer, socketPath, cleanup := newMockServer(t)
	defer cleanup()

	cs := &controllerServer{
		nodeID:           "test-node",
		kubeEdgeEndpoint: socketPath,
	}

	mockServer.response = &model.Message{
		Content: "invalid-base64-content",
	}

	req := &csi.DeleteVolumeRequest{
		VolumeId: "test-volume",
	}
	resp, err := cs.DeleteVolume(context.Background(), req)
	assert.Error(err)
	assert.Nil(resp)

	mockServer.response = &model.Message{
		Content: 123,
	}

	resp, err = cs.DeleteVolume(context.Background(), req)
	assert.Error(err)
	assert.Nil(resp)
	assert.Contains(err.Error(), "content type")
}

type mockErrorServer struct {
	t            *testing.T
	responseType string
	listener     net.Listener
}

func newMockErrorServer(t *testing.T, responseType string) (string, func()) {
	dir, err := os.MkdirTemp("", "csi-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}

	socketPath := filepath.Join(dir, "test.sock")
	listener, err := net.Listen("unix", socketPath)
	if err != nil {
		t.Fatalf("Failed to create Unix domain socket: %v", err)
	}

	server := &mockErrorServer{
		t:            t,
		responseType: responseType,
		listener:     listener,
	}

	go server.serve()

	cleanup := func() {
		listener.Close()
		os.RemoveAll(dir)
	}

	return "unix://" + socketPath, cleanup
}

func (m *mockErrorServer) serve() {
	for {
		conn, err := m.listener.Accept()
		if err != nil {
			return
		}
		go m.handleConnection(conn)
	}
}

func (m *mockErrorServer) handleConnection(conn net.Conn) {
	defer conn.Close()

	buf := make([]byte, DefaultBufferSize)
	_, err := conn.Read(buf)
	if err != nil {
		m.t.Errorf("Failed to read from connection: %v", err)
		return
	}

	var response []byte
	switch m.responseType {
	case "invalid_json":
		response = []byte(`{invalid json`)
	case "non_string_content":
		msg := &model.Message{
			Content: float64(123),
		}
		response, _ = json.Marshal(msg)
	case "invalid_base64":
		msg := &model.Message{
			Content: "invalid base64",
		}
		response, _ = json.Marshal(msg)
	case "error_operation":
		msg := &model.Message{
			Router: model.MessageRoute{
				Operation: model.ResponseErrorOperation,
			},
			Content: "test error",
		}
		response, _ = json.Marshal(msg)
	default:
		msg := &model.Message{
			Content: base64.StdEncoding.EncodeToString([]byte(`{}`)),
		}
		response, _ = json.Marshal(msg)
	}

	if _, err := conn.Write(response); err != nil {
		m.t.Errorf("Failed to write response: %v", err)
		return
	}
}

func TestResourceBuildError(t *testing.T) {
	assert := assert.New(t)

	cs := &controllerServer{
		nodeID: "",
	}

	req := &csi.CreateVolumeRequest{
		Name: "test-volume",
		VolumeCapabilities: []*csi.VolumeCapability{
			{
				AccessType: &csi.VolumeCapability_Mount{},
			},
		},
	}

	resp, err := cs.CreateVolume(context.Background(), req)
	assert.Error(err)
	assert.Nil(resp)
	assert.Contains(err.Error(), "required parameter are not set")
}

func TestDeleteVolumeErrors(t *testing.T) {
	assert := assert.New(t)

	testCases := []struct {
		name          string
		responseType  string
		errorContains string
	}{
		{
			name:          "Invalid JSON response",
			responseType:  "invalid_json",
			errorContains: "invalid character 'i' looking for beginning of object key string",
		},
		{
			name:          "Non-string content",
			responseType:  "non_string_content",
			errorContains: fmt.Sprintf("content type %T is not string", float64(0)),
		},
		{
			name:          "Invalid base64",
			responseType:  "invalid_base64",
			errorContains: "illegal base64 data",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			socketPath, cleanup := newMockErrorServer(t, tc.responseType)
			defer cleanup()

			cs := &controllerServer{
				nodeID:           "test-node",
				kubeEdgeEndpoint: socketPath,
			}

			req := &csi.DeleteVolumeRequest{
				VolumeId: "test-volume",
			}

			resp, err := cs.DeleteVolume(context.Background(), req)
			assert.Error(err)
			assert.Nil(resp)
			assert.Contains(err.Error(), tc.errorContains)
		})
	}
}

func TestControllerPublishVolumeErrors(t *testing.T) {
	assert := assert.New(t)

	testCases := []struct {
		name          string
		responseType  string
		errorContains string
	}{
		{
			name:          "Invalid JSON response",
			responseType:  "invalid_json",
			errorContains: "invalid character 'i' looking for beginning of object key string",
		},
		{
			name:          "Non-string content",
			responseType:  "non_string_content",
			errorContains: fmt.Sprintf("content type %T is not string", float64(0)),
		},
		{
			name:          "Invalid base64",
			responseType:  "invalid_base64",
			errorContains: "illegal base64 data",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			socketPath, cleanup := newMockErrorServer(t, tc.responseType)
			defer cleanup()

			cs := &controllerServer{
				nodeID:           "test-node",
				kubeEdgeEndpoint: socketPath,
			}

			req := &csi.ControllerPublishVolumeRequest{
				VolumeId: "test-volume",
				NodeId:   "test-node",
				VolumeCapability: &csi.VolumeCapability{
					AccessType: &csi.VolumeCapability_Mount{},
				},
			}

			resp, err := cs.ControllerPublishVolume(context.Background(), req)
			assert.Error(err)
			assert.Nil(resp)
			assert.Contains(err.Error(), tc.errorContains)
		})
	}
}

func TestControllerUnpublishVolumeErrors(t *testing.T) {
	assert := assert.New(t)

	testCases := []struct {
		name          string
		responseType  string
		errorContains string
	}{
		{
			name:          "Invalid JSON response",
			responseType:  "invalid_json",
			errorContains: "invalid character 'i' looking for beginning of object key string",
		},
		{
			name:          "Non-string content",
			responseType:  "non_string_content",
			errorContains: fmt.Sprintf("content type %T is not string", float64(0)),
		},
		{
			name:          "Invalid base64",
			responseType:  "invalid_base64",
			errorContains: "illegal base64 data",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			socketPath, cleanup := newMockErrorServer(t, tc.responseType)
			defer cleanup()

			cs := &controllerServer{
				nodeID:           "test-node",
				kubeEdgeEndpoint: socketPath,
			}

			req := &csi.ControllerUnpublishVolumeRequest{
				VolumeId: "test-volume",
				NodeId:   "test-node",
			}

			resp, err := cs.ControllerUnpublishVolume(context.Background(), req)
			assert.Error(err)
			assert.Nil(resp)
			assert.Contains(err.Error(), tc.errorContains)
		})
	}
}
