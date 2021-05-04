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

package csidriver

import (
	"encoding/base64"
	"encoding/json"
	"errors"

	"github.com/container-storage-interface/spec/lib/go/csi"
	"github.com/golang/protobuf/jsonpb"
	"github.com/google/uuid"
	"golang.org/x/net/context"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"k8s.io/klog/v2"

	"github.com/kubeedge/beehive/pkg/core/model"
	"github.com/kubeedge/kubeedge/common/constants"
)

type controllerServer struct {
	caps             []*csi.ControllerServiceCapability
	nodeID           string
	kubeEdgeEndpoint string
}

// newControllerServer creates controller server
func newControllerServer(nodeID, kubeEdgeEndpoint string) *controllerServer {
	return &controllerServer{
		caps: getControllerServiceCapabilities(
			[]csi.ControllerServiceCapability_RPC_Type{
				csi.ControllerServiceCapability_RPC_CREATE_DELETE_VOLUME,
				csi.ControllerServiceCapability_RPC_PUBLISH_UNPUBLISH_VOLUME,
			}),
		nodeID:           nodeID,
		kubeEdgeEndpoint: kubeEdgeEndpoint,
	}
}

// CreateVolume issues create volume func
func (cs *controllerServer) CreateVolume(ctx context.Context, req *csi.CreateVolumeRequest) (*csi.CreateVolumeResponse, error) {
	// Check arguments
	if len(req.GetName()) == 0 {
		return nil, status.Error(codes.InvalidArgument, "Name missing in request")
	}
	caps := req.GetVolumeCapabilities()
	if caps == nil {
		return nil, status.Error(codes.InvalidArgument, "Volume Capabilities missing in request")
	}

	volumeID := uuid.New().String()

	// Build message struct
	msg := model.NewMessage("")
	resource, err := buildResource(cs.nodeID,
		DefaultNamespace,
		constants.CSIResourceTypeVolume,
		volumeID)
	if err != nil {
		klog.Errorf("build message resource failed with error: %s", err)
		return nil, err
	}

	m := jsonpb.Marshaler{}
	js, err := m.MarshalToString(req)
	if err != nil {
		klog.Errorf("failed to marshal to string with error: %s", err)
		return nil, err
	}
	klog.V(4).Infof("create volume marshal to string: %s", js)
	msg.Content = js
	msg.BuildRouter(DefaultReceiveModuleName,
		GroupResource,
		resource,
		constants.CSIOperationTypeCreateVolume)

	// Marshal message
	reqData, err := json.Marshal(msg)
	if err != nil {
		klog.Errorf("marshal request failed with error: %v", err)
		return nil, err
	}

	// Send message to KubeEdge
	resdata, err := sendToKubeEdge(string(reqData), cs.kubeEdgeEndpoint)
	if err != nil {
		klog.Errorf("send to kubeedge failed with error: %v", err)
		return nil, err
	}

	// Unmarshal message
	result, err := extractMessage(resdata)
	if err != nil {
		klog.Errorf("unmarshal response failed with error: %v", err)
		return nil, err
	}

	klog.V(4).Infof("create volume result: %v", result)
	data := result.GetContent().(string)

	if result.GetOperation() == model.ResponseErrorOperation {
		klog.Errorf("create volume with error: %s", data)
		return nil, errors.New(data)
	}

	decodeBytes, err := base64.StdEncoding.DecodeString(data)
	if err != nil {
		klog.Errorf("create volume decode with error: %v", err)
		return nil, err
	}

	response := &csi.CreateVolumeResponse{}
	err = json.Unmarshal([]byte(decodeBytes), response)
	if err != nil {
		klog.Errorf("create volume unmarshal with error: %v", err)
		return nil, nil
	}
	klog.V(4).Infof("create volume response: %v", response)

	createVolumeResponse := &csi.CreateVolumeResponse{
		Volume: &csi.Volume{
			VolumeId:      response.Volume.VolumeId,
			CapacityBytes: req.GetCapacityRange().GetRequiredBytes(),
			VolumeContext: req.GetParameters(),
		},
	}
	if req.GetVolumeContentSource() != nil {
		createVolumeResponse.Volume.ContentSource = req.GetVolumeContentSource()
	}
	return createVolumeResponse, nil
}

// DeleteVolume issues delete volume func
func (cs *controllerServer) DeleteVolume(ctx context.Context, req *csi.DeleteVolumeRequest) (*csi.DeleteVolumeResponse, error) {
	// Check arguments
	if len(req.GetVolumeId()) == 0 {
		return nil, status.Error(codes.InvalidArgument, "Volume ID missing in request")
	}

	// Build message struct
	msg := model.NewMessage("")
	resource, err := buildResource(cs.nodeID,
		DefaultNamespace,
		constants.CSIResourceTypeVolume,
		req.GetVolumeId())
	if err != nil {
		klog.Errorf("build message resource failed with error: %s", err)
		return nil, err
	}

	m := jsonpb.Marshaler{}
	js, err := m.MarshalToString(req)
	if err != nil {
		klog.Errorf("failed to marshal to string with error: %s", err)
		return nil, err
	}
	klog.V(4).Infof("delete volume marshal to string: %s", js)
	msg.Content = js
	msg.BuildRouter(DefaultReceiveModuleName,
		GroupResource,
		resource,
		constants.CSIOperationTypeDeleteVolume)

	// Marshal message
	reqData, err := json.Marshal(msg)
	if err != nil {
		klog.Errorf("marshal request failed with error: %v", err)
		return nil, err
	}

	// Send message to KubeEdge
	resdata, err := sendToKubeEdge(string(reqData), cs.kubeEdgeEndpoint)
	if err != nil {
		klog.Errorf("send to kubeedge failed with error: %v", err)
		return nil, err
	}

	// Unmarshal message
	result, err := extractMessage(resdata)
	if err != nil {
		klog.Errorf("unmarshal response failed with error: %v", err)
		return nil, err
	}

	klog.V(4).Infof("delete volume result: %v", result)
	data := result.GetContent().(string)

	if msg.GetOperation() == model.ResponseErrorOperation {
		klog.Errorf("delete volume with error: %s", data)
		return nil, errors.New(data)
	}

	decodeBytes, err := base64.StdEncoding.DecodeString(data)
	if err != nil {
		klog.Errorf("delete volume decode with error: %v", err)
		return nil, err
	}

	deleteVolumeResponse := &csi.DeleteVolumeResponse{}
	err = json.Unmarshal([]byte(decodeBytes), deleteVolumeResponse)
	if err != nil {
		klog.Errorf("delete volume unmarshal with error: %v", err)
		return nil, nil
	}
	klog.V(4).Infof("delete volume response: %v", deleteVolumeResponse)
	return deleteVolumeResponse, nil
}

// ControllerPublishVolume issues controller publish volume func
func (cs *controllerServer) ControllerPublishVolume(ctx context.Context, req *csi.ControllerPublishVolumeRequest) (*csi.ControllerPublishVolumeResponse, error) {
	instanceID := req.GetNodeId()
	volumeID := req.GetVolumeId()

	if len(volumeID) == 0 {
		return nil, status.Error(codes.InvalidArgument, "ControllerPublishVolume Volume ID must be provided")
	}

	if len(instanceID) == 0 {
		return nil, status.Error(codes.InvalidArgument, "ControllerPublishVolume Instance ID must be provided")
	}

	// Build message struct
	msg := model.NewMessage("")
	resource, err := buildResource(cs.nodeID,
		DefaultNamespace,
		constants.CSIResourceTypeVolume,
		volumeID)
	if err != nil {
		klog.Errorf("build message resource failed with error: %s", err)
		return nil, err
	}

	m := jsonpb.Marshaler{}
	js, err := m.MarshalToString(req)
	if err != nil {
		klog.Errorf("failed to marshal to string with error: %s", err)
		return nil, err
	}
	klog.V(4).Infof("controller publish volume marshal to string: %s", js)
	msg.Content = js
	msg.BuildRouter(DefaultReceiveModuleName,
		GroupResource,
		resource,
		constants.CSIOperationTypeControllerPublishVolume)

	// Marshal message
	reqData, err := json.Marshal(msg)
	if err != nil {
		klog.Errorf("marshal request failed with error: %v", err)
		return nil, err
	}

	// Send message to KubeEdge
	resdata, err := sendToKubeEdge(string(reqData), cs.kubeEdgeEndpoint)
	if err != nil {
		klog.Errorf("send to kubeedge failed with error: %v", err)
		return nil, err
	}

	// Unmarshal message
	result, err := extractMessage(resdata)
	if err != nil {
		klog.Errorf("unmarshal response failed with error: %v", err)
		return nil, err
	}

	klog.V(4).Infof("controller publish volume result: %v", result)
	data := result.GetContent().(string)

	if msg.GetOperation() == model.ResponseErrorOperation {
		klog.Errorf("controller publish volume with error: %s", data)
		return nil, errors.New(data)
	}

	decodeBytes, err := base64.StdEncoding.DecodeString(data)
	if err != nil {
		klog.Errorf("controller publish volume decode with error: %v", err)
		return nil, err
	}

	controllerPublishVolumeResponse := &csi.ControllerPublishVolumeResponse{}
	err = json.Unmarshal([]byte(decodeBytes), controllerPublishVolumeResponse)
	if err != nil {
		klog.Errorf("controller publish volume unmarshal with error: %v", err)
		return nil, nil
	}
	klog.V(4).Infof("controller publish volume response: %v", controllerPublishVolumeResponse)
	return controllerPublishVolumeResponse, nil
}

// ControllerUnpublishVolume issues controller unpublish volume func
func (cs *controllerServer) ControllerUnpublishVolume(ctx context.Context, req *csi.ControllerUnpublishVolumeRequest) (*csi.ControllerUnpublishVolumeResponse, error) {
	instanceID := req.GetNodeId()
	volumeID := req.GetVolumeId()

	if len(volumeID) == 0 {
		return nil, status.Error(codes.InvalidArgument, "ControllerUnpublishVolume Volume ID must be provided")
	}

	if len(instanceID) == 0 {
		return nil, status.Error(codes.InvalidArgument, "ControllerUnpublishVolume Instance ID must be provided")
	}

	// Build message struct
	msg := model.NewMessage("")
	resource, err := buildResource(cs.nodeID,
		DefaultNamespace,
		constants.CSIResourceTypeVolume,
		volumeID)
	if err != nil {
		klog.Errorf("Build message resource failed with error: %s", err)
		return nil, err
	}

	m := jsonpb.Marshaler{}
	js, err := m.MarshalToString(req)
	if err != nil {
		klog.Errorf("failed to marshal to string with error: %s", err)
		return nil, err
	}
	klog.V(4).Infof("controller Unpublish Volume marshal to string: %s", js)
	msg.Content = js
	msg.BuildRouter(DefaultReceiveModuleName,
		GroupResource,
		resource,
		constants.CSIOperationTypeControllerUnpublishVolume)

	// Marshal message
	reqData, err := json.Marshal(msg)
	if err != nil {
		klog.Errorf("marshal request failed with error: %v", err)
		return nil, err
	}

	// Send message to KubeEdge
	resdata, err := sendToKubeEdge(string(reqData), cs.kubeEdgeEndpoint)
	if err != nil {
		klog.Errorf("send to kubeedge failed with error: %v", err)
		return nil, err
	}

	// Unmarshal message
	result, err := extractMessage(resdata)
	if err != nil {
		klog.Errorf("unmarshal response failed with error: %v", err)
		return nil, err
	}

	klog.V(4).Infof("controller Unpublish Volume result: %v", result)
	data := result.GetContent().(string)

	if msg.GetOperation() == model.ResponseErrorOperation {
		klog.Errorf("controller Unpublish Volume with error: %s", data)
		return nil, errors.New(data)
	}

	decodeBytes, err := base64.StdEncoding.DecodeString(data)
	if err != nil {
		klog.Errorf("controller Unpublish Volume decode with error: %v", err)
		return nil, err
	}

	controllerUnpublishVolumeResponse := &csi.ControllerUnpublishVolumeResponse{}
	err = json.Unmarshal([]byte(decodeBytes), controllerUnpublishVolumeResponse)
	if err != nil {
		klog.Errorf("controller Unpublish Volume unmarshal with error: %v", err)
		return nil, nil
	}
	klog.V(4).Infof("controller Unpublish Volume response: %v", controllerUnpublishVolumeResponse)
	return controllerUnpublishVolumeResponse, nil
}

func (cs *controllerServer) ValidateVolumeCapabilities(ctx context.Context, req *csi.ValidateVolumeCapabilitiesRequest) (*csi.ValidateVolumeCapabilitiesResponse, error) {
	// Check arguments
	if len(req.GetVolumeId()) == 0 {
		return nil, status.Error(codes.InvalidArgument, "Volume ID cannot be empty")
	}
	if len(req.VolumeCapabilities) == 0 {
		return nil, status.Error(codes.InvalidArgument, req.VolumeId)
	}

	for _, cap := range req.GetVolumeCapabilities() {
		if cap.GetMount() == nil && cap.GetBlock() == nil {
			return nil, status.Error(codes.InvalidArgument, "cannot have both mount and block access type be undefined")
		}
	}

	return &csi.ValidateVolumeCapabilitiesResponse{
		Confirmed: &csi.ValidateVolumeCapabilitiesResponse_Confirmed{
			VolumeContext:      req.GetVolumeContext(),
			VolumeCapabilities: req.GetVolumeCapabilities(),
			Parameters:         req.GetParameters(),
		},
	}, nil
}

func (cs *controllerServer) ControllerGetCapabilities(ctx context.Context, req *csi.ControllerGetCapabilitiesRequest) (*csi.ControllerGetCapabilitiesResponse, error) {
	return &csi.ControllerGetCapabilitiesResponse{
		Capabilities: cs.caps,
	}, nil
}

func getControllerServiceCapabilities(cl []csi.ControllerServiceCapability_RPC_Type) []*csi.ControllerServiceCapability {
	var csc []*csi.ControllerServiceCapability

	for _, cap := range cl {
		klog.V(4).Infof("Enabling controller service capability: %v", cap.String())
		csc = append(csc, &csi.ControllerServiceCapability{
			Type: &csi.ControllerServiceCapability_Rpc{
				Rpc: &csi.ControllerServiceCapability_RPC{
					Type: cap,
				},
			},
		})
	}

	return csc
}

func (cs *controllerServer) GetCapacity(ctx context.Context, req *csi.GetCapacityRequest) (*csi.GetCapacityResponse, error) {
	return nil, status.Error(codes.Unimplemented, "")
}

func (cs *controllerServer) ListVolumes(ctx context.Context, req *csi.ListVolumesRequest) (*csi.ListVolumesResponse, error) {
	return nil, status.Error(codes.Unimplemented, "")
}

func (cs *controllerServer) ControllerExpandVolume(ctx context.Context, req *csi.ControllerExpandVolumeRequest) (*csi.ControllerExpandVolumeResponse, error) {
	return nil, status.Error(codes.Unimplemented, "ControllerExpandVolume is not yet implemented")
}

func (cs *controllerServer) CreateSnapshot(ctx context.Context, req *csi.CreateSnapshotRequest) (*csi.CreateSnapshotResponse, error) {
	return nil, status.Error(codes.Unimplemented, "CreateSnapshot is not yet implemented")
}

func (cs *controllerServer) DeleteSnapshot(ctx context.Context, req *csi.DeleteSnapshotRequest) (*csi.DeleteSnapshotResponse, error) {
	return nil, status.Error(codes.Unimplemented, "DeleteSnapshot is not yet implemented")
}

func (cs *controllerServer) ListSnapshots(ctx context.Context, req *csi.ListSnapshotsRequest) (*csi.ListSnapshotsResponse, error) {
	return nil, status.Error(codes.Unimplemented, "ListSnapshots is not yet implemented")
}
