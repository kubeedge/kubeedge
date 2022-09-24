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

package csi

import (
	"context"

	"github.com/container-storage-interface/spec/lib/go/csi"
	"k8s.io/klog/v2"
)

type Controller struct {
	csiClient csiClient
}

func (c *Controller) CreateVolume(req *csi.CreateVolumeRequest) (*csi.CreateVolumeResponse, error) {
	if c.csiClient == nil {
		var err error
		c.csiClient, err = newCsiDriverClient(csiDriverName("csi-hostpath"))
		if err != nil {
			klog.Errorf("failed to create newCsiDriverClient: %v", err)
			return nil, err
		}
	}
	client := c.csiClient

	ctx, cancel := context.WithTimeout(context.Background(), csiTimeout)
	defer cancel()

	res, err := client.ControllerGetCapabilities(ctx, &csi.ControllerGetCapabilitiesRequest{})
	if err != nil {
		klog.Errorf("failed to ControllerGetCapabilities: %v", err)
		return nil, err
	}
	for _, cap := range res.Capabilities {
		if csi.ControllerServiceCapability_RPC_CREATE_DELETE_VOLUME == cap.GetRpc().GetType() {
			return client.CreateVolume(ctx, req)
		}
	}

	return &csi.CreateVolumeResponse{}, nil
}

func (c *Controller) DeleteVolume(req *csi.DeleteVolumeRequest) (*csi.DeleteVolumeResponse, error) {
	if c.csiClient == nil {
		var err error
		c.csiClient, err = newCsiDriverClient(csiDriverName("csi-hostpath"))
		if err != nil {
			klog.Errorf("failed to create newCsiDriverClient: %v", err)
			return nil, err
		}
	}
	client := c.csiClient

	ctx, cancel := context.WithTimeout(context.Background(), csiTimeout)
	defer cancel()

	res, err := client.ControllerGetCapabilities(ctx, &csi.ControllerGetCapabilitiesRequest{})
	if err != nil {
		klog.Errorf("failed to ControllerGetCapabilities: %v", err)
		return nil, err
	}
	for _, cap := range res.Capabilities {
		if csi.ControllerServiceCapability_RPC_CREATE_DELETE_VOLUME == cap.GetRpc().GetType() {
			return client.DeleteVolume(ctx, req)
		}
	}

	return &csi.DeleteVolumeResponse{}, nil
}

func (c *Controller) ControllerPublishVolume(req *csi.ControllerPublishVolumeRequest) (*csi.ControllerPublishVolumeResponse, error) {
	if c.csiClient == nil {
		var err error
		c.csiClient, err = newCsiDriverClient(csiDriverName("csi-hostpath"))
		if err != nil {
			klog.Errorf("failed to create newCsiDriverClient: %v", err)
			return nil, err
		}
	}
	client := c.csiClient

	ctx, cancel := context.WithTimeout(context.Background(), csiTimeout)
	defer cancel()

	res, err := client.ControllerGetCapabilities(ctx, &csi.ControllerGetCapabilitiesRequest{})
	if err != nil {
		klog.Errorf("failed to ControllerGetCapabilities: %v", err)
		return nil, err
	}
	for _, cap := range res.Capabilities {
		if csi.ControllerServiceCapability_RPC_PUBLISH_UNPUBLISH_VOLUME == cap.GetRpc().GetType() {
			return client.ControllerPublishVolume(ctx, req)
		}
	}

	return &csi.ControllerPublishVolumeResponse{}, nil
}

func (c *Controller) ControllerUnpublishVolume(req *csi.ControllerUnpublishVolumeRequest) (*csi.ControllerUnpublishVolumeResponse, error) {
	if c.csiClient == nil {
		var err error
		c.csiClient, err = newCsiDriverClient(csiDriverName("csi-hostpath"))
		if err != nil {
			klog.Errorf("failed to create newCsiDriverClient: %v", err)
			return nil, err
		}
	}
	client := c.csiClient

	ctx, cancel := context.WithTimeout(context.Background(), csiTimeout)
	defer cancel()

	res, err := client.ControllerGetCapabilities(ctx, &csi.ControllerGetCapabilitiesRequest{})
	if err != nil {
		klog.Errorf("failed to ControllerGetCapabilities: %v", err)
		return nil, err
	}
	for _, cap := range res.Capabilities {
		if csi.ControllerServiceCapability_RPC_PUBLISH_UNPUBLISH_VOLUME == cap.GetRpc().GetType() {
			return client.ControllerUnpublishVolume(ctx, req)
		}
	}

	return &csi.ControllerUnpublishVolumeResponse{}, nil
}
