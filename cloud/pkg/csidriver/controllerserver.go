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
	"fmt"

	"github.com/container-storage-interface/spec/lib/go/csi"
	"golang.org/x/net/context"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"k8s.io/klog/v2"

	"github.com/kubeedge/kubeedge/cloud/pkg/csidriver/state"
	"github.com/kubeedge/kubeedge/common/constants"
)

type controllerServer struct {
	caps             []*csi.ControllerServiceCapability
	sendFn           KubeEdgeSendFn
	kubeEdgeEndpoint string
	inFlight         *inFlight
	store            *state.Store
	topologyKey      string
}

type KubeEdgeSendFn func(req interface{}, nodeID, volumeID, csiOp string, res interface{}, kubeEdgeEndpoint string) error

// newControllerServer creates controller server
func newControllerServer(kubeEdgeEndpoint string, store *state.Store, topologyKey string) *controllerServer {
	return &controllerServer{
		caps: getControllerServiceCapabilities(
			[]csi.ControllerServiceCapability_RPC_Type{
				csi.ControllerServiceCapability_RPC_CREATE_DELETE_VOLUME,
				csi.ControllerServiceCapability_RPC_PUBLISH_UNPUBLISH_VOLUME,
			}),
		sendFn:           sendToKubeEdge,
		kubeEdgeEndpoint: kubeEdgeEndpoint,
		topologyKey:      topologyKey,
		inFlight:         newInFlight(),
		store:            store,
	}
}

// CreateVolume issues create volume func
func (cs *controllerServer) CreateVolume(ctx context.Context, req *csi.CreateVolumeRequest) (*csi.CreateVolumeResponse, error) {
	volName := req.GetName()
	if len(volName) == 0 {
		return nil, status.Error(codes.InvalidArgument, "Name missing in request")
	}
	caps := req.GetVolumeCapabilities()
	if caps == nil {
		return nil, status.Error(codes.InvalidArgument, "Volume Capabilities missing in request")
	}
	klog.V(4).Infof("create volume request id=%s req=%#v", volName, req)

	if ok := cs.inFlight.Insert(volName); !ok {
		return nil, status.Errorf(codes.Aborted, "request already inflight for %s", volName)
	}
	defer cs.inFlight.Delete(volName)

	edgeNode, err := pickEdgeNode(req.GetAccessibilityRequirements(), cs.topologyKey)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "Can not pick edge node based on accessibility requirements")
	}
	klog.V(4).Infof("picked edge node: %s", edgeNode)

	// Send message to KubeEdge
	res := &csi.CreateVolumeResponse{}
	err = cs.sendFn(req, edgeNode, volName, constants.CSIOperationTypeCreateVolume, res, cs.kubeEdgeEndpoint)
	if err != nil {
		klog.Errorf("send to kubeedge failed with error: %v", err)
		return nil, err
	}

	err = cs.store.Update(res.Volume.VolumeId, edgeNode)
	if err != nil {
		return nil, status.Error(codes.Internal, "unable to update state")
	}

	klog.V(4).Infof("create volume response: %#v", res)
	createVolumeResponse := &csi.CreateVolumeResponse{
		Volume: &csi.Volume{
			VolumeId:      res.Volume.VolumeId,
			CapacityBytes: req.GetCapacityRange().GetRequiredBytes(),
		},
	}
	if req.GetVolumeContentSource() != nil {
		createVolumeResponse.Volume.ContentSource = req.GetVolumeContentSource()
	}
	klog.V(4).Infof("returning volume response: %#v", createVolumeResponse)
	return createVolumeResponse, nil
}

func pickEdgeNode(requirement *csi.TopologyRequirement, topoKey string) (string, error) {
	klog.Info("topology requirements: %#v", requirement)
	if requirement == nil {
		return "", fmt.Errorf("missing topology requirements")
	}
	for _, topology := range requirement.GetPreferred() {
		node, exists := findTopoKey(topology.GetSegments(), topoKey)
		if exists {
			return node, nil
		}
	}
	for _, topology := range requirement.GetRequisite() {
		node, exists := findTopoKey(topology.GetSegments(), topoKey)
		if exists {
			return node, nil
		}
	}
	return "", fmt.Errorf("could not find matching node")
}

func findTopoKey(segments map[string]string, topoKey string) (string, bool) {
	node, exists := segments[topoKey]
	if exists {
		return node, true
	}
	return "", false
}

// DeleteVolume issues delete volume func
func (cs *controllerServer) DeleteVolume(ctx context.Context, req *csi.DeleteVolumeRequest) (*csi.DeleteVolumeResponse, error) {
	volumeID := req.GetVolumeId()
	if len(volumeID) == 0 {
		return nil, status.Error(codes.InvalidArgument, "Volume ID missing in request")
	}
	edgeName, err := cs.store.Get(volumeID)
	if err != nil {
		return nil, status.Error(codes.Internal, "unable to get state")
	}
	klog.V(4).Infof("delete volume %s from %s", volumeID, edgeName)
	res := &csi.DeleteVolumeResponse{}
	err = cs.sendFn(req, edgeName, volumeID, constants.CSIOperationTypeDeleteVolume, res, cs.kubeEdgeEndpoint)
	if err != nil {
		klog.Errorf("send to kubeedge failed with error: %v", err)
		return nil, err
	}
	err = cs.store.Delete(volumeID)
	if err != nil {
		return nil, status.Error(codes.Internal, fmt.Sprintf("unable to update state: %s", err))
	}
	klog.V(4).Infof("delete volume response: %v", res)
	return res, nil
}

// ControllerPublishVolume issues controller publish volume func
func (cs *controllerServer) ControllerPublishVolume(ctx context.Context, req *csi.ControllerPublishVolumeRequest) (*csi.ControllerPublishVolumeResponse, error) {
	volumeID := req.GetVolumeId()
	if len(volumeID) == 0 {
		return nil, status.Error(codes.InvalidArgument, "ControllerPublishVolume Volume ID must be provided")
	}
	edgeName := req.GetNodeId()
	if edgeName == "" {
		return nil, status.Error(codes.InvalidArgument, "ControllerPublishVolume Node ID must be provided")
	}
	klog.V(4).Infof("publish volume %s on %s", volumeID, edgeName)
	res := &csi.ControllerPublishVolumeResponse{}
	err := cs.sendFn(req, edgeName, volumeID, constants.CSIOperationTypeControllerPublishVolume, res, cs.kubeEdgeEndpoint)
	if err != nil {
		klog.Errorf("send to kubeedge failed with error: %v", err)
		return nil, err
	}
	klog.V(4).Infof("controller publish volume response: %v", res)
	return res, nil
}

// ControllerUnpublishVolume issues controller unpublish volume func
func (cs *controllerServer) ControllerUnpublishVolume(ctx context.Context, req *csi.ControllerUnpublishVolumeRequest) (*csi.ControllerUnpublishVolumeResponse, error) {
	volumeID := req.GetVolumeId()
	if len(volumeID) == 0 {
		return nil, status.Error(codes.InvalidArgument, "ControllerUnpublishVolume Volume ID must be provided")
	}
	edgeName := req.GetNodeId()
	if edgeName == "" {
		return nil, status.Error(codes.InvalidArgument, "ControllerUnpublishVolume Node ID must be provided")
	}
	klog.V(4).Infof("unpublish volume %s on %s", volumeID, edgeName)
	res := &csi.ControllerUnpublishVolumeResponse{}
	err := cs.sendFn(req, edgeName, volumeID, constants.CSIOperationTypeControllerUnpublishVolume, res, cs.kubeEdgeEndpoint)
	if err != nil {
		klog.Errorf("send to kubeedge failed with error: %v", err)
		return nil, err
	}
	klog.V(4).Infof("controller Unpublish Volume response: %v", res)
	return res, nil
}

func (cs *controllerServer) ValidateVolumeCapabilities(ctx context.Context, req *csi.ValidateVolumeCapabilitiesRequest) (*csi.ValidateVolumeCapabilitiesResponse, error) {
	if len(req.GetVolumeId()) == 0 {
		return nil, status.Error(codes.InvalidArgument, "Volume ID cannot be empty")
	}
	if len(req.VolumeCapabilities) == 0 {
		return nil, status.Error(codes.InvalidArgument, "Volume Capabilities can not be empty")
	}

	for _, cap := range req.GetVolumeCapabilities() {
		if cap.GetMount() == nil && cap.GetBlock() == nil {
			return nil, status.Error(codes.InvalidArgument, "Cannot have both mount and block access type be undefined")
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

func (cs *controllerServer) ControllerGetVolume(context.Context, *csi.ControllerGetVolumeRequest) (*csi.ControllerGetVolumeResponse, error) {
	return nil, status.Error(codes.Unimplemented, "ControllerGetVolume is not yet implemented")
}
