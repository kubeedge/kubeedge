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

package client

import (
	"context"
	"fmt"

	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/rest"

	"github.com/kubeedge/api/apis/devices/v1beta1"
	"github.com/kubeedge/api/client/clientset/versioned/scheme"
	"github.com/kubeedge/kubeedge/keadm/cmd/keadm/app/cmd/common"
)

type DeviceRequest struct {
	Namespace     string
	DeviceName    string
	LabelSelector string
	AllNamespaces bool
}

func (deviceRequest *DeviceRequest) GetDevice(ctx context.Context) (*v1beta1.Device, error) {
	versionedClient, err := VersionedKubeClient()
	if err != nil {
		return nil, err
	}
	device, err := versionedClient.DevicesV1beta1().Devices(deviceRequest.Namespace).Get(ctx, deviceRequest.DeviceName, metaV1.GetOptions{})
	if err != nil {
		return nil, err
	}
	device.APIVersion = common.DeviceAPIVersion
	device.Kind = common.DeviceKind
	return device, nil
}

func (deviceRequest *DeviceRequest) GetDevices(ctx context.Context) (*v1beta1.DeviceList, error) {
	versionedClient, err := VersionedKubeClient()
	if err != nil {
		return nil, err
	}
	if deviceRequest.AllNamespaces {
		deviceList, err := versionedClient.DevicesV1beta1().Devices(metaV1.NamespaceAll).List(ctx, metaV1.ListOptions{
			LabelSelector: deviceRequest.LabelSelector,
		})
		if err != nil {
			return nil, fmt.Errorf("failed to list devices: %v", err)
		}
		return deviceList, nil
	}

	deviceList, err := versionedClient.DevicesV1beta1().Devices(deviceRequest.Namespace).List(ctx, metaV1.ListOptions{
		LabelSelector: deviceRequest.LabelSelector,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list devices: %v", err)
	}
	return deviceList, nil
}

func (deviceRequest *DeviceRequest) UpdateDevice(ctx context.Context, device *v1beta1.Device) (*rest.Result, error) {
	versionedClient, err := VersionedKubeClient()
	if err != nil {
		return nil, err
	}

	res := versionedClient.DevicesV1beta1().RESTClient().
		Put().
		Namespace(deviceRequest.Namespace).
		Resource("devices").
		Name(device.Name).
		VersionedParams(&metaV1.UpdateOptions{}, scheme.ParameterCodec).
		Body(device).
		Do(ctx)
	return &res, nil
}
