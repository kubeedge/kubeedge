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

package controller

import (
	"testing"

	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/kubeedge/api/apis/devices/v1beta1"
	crdfake "github.com/kubeedge/api/client/clientset/versioned/fake"
	"github.com/kubeedge/beehive/pkg/core/model"
	"github.com/kubeedge/kubeedge/cloud/pkg/devicecontroller/manager"
	"github.com/kubeedge/kubeedge/pkg/util"
)

// fakeMessageLayer is a minimal MessageLayer that records sent messages.
type fakeMessageLayer struct {
	sent []model.Message
}

func (f *fakeMessageLayer) Send(message model.Message) error {
	f.sent = append(f.sent, message)
	return nil
}

func (f *fakeMessageLayer) Receive() (model.Message, error) {
	return model.Message{}, nil
}

func (f *fakeMessageLayer) Response(message model.Message) error {
	return nil
}

func TestDeviceUpdatedInvalidCachedType(t *testing.T) {
	t.Run("corrupted cache entry is replaced with new device (no NodeName)", func(t *testing.T) {
		dc := &DownstreamController{
			deviceManager:       &manager.DeviceManager{},
			deviceStatusManager: &manager.DeviceStatusManager{},
			crdClient:           crdfake.NewSimpleClientset(),
		}
		device := &v1beta1.Device{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: "default",
				Name:      "dev1",
			},
		}
		deviceID := util.GetResourceID(device.Namespace, device.Name)
		// Simulate a corrupted cache entry of the wrong type.
		dc.deviceManager.Device.Store(deviceID, "not-a-device")

		assert.NotPanics(t, func() {
			dc.deviceUpdated(device)
		})

		// After recovery, the cache must hold the correct *v1beta1.Device.
		val, ok := dc.deviceManager.Device.Load(deviceID)
		assert.True(t, ok, "device should be present in cache after recovery")
		_, isDevice := val.(*v1beta1.Device)
		assert.True(t, isDevice, "cached value should be a *v1beta1.Device after recovery")
	})

	t.Run("corrupted cache entry triggers deviceAdded downlink when NodeName is set", func(t *testing.T) {
		fml := &fakeMessageLayer{}
		dc := &DownstreamController{
			deviceManager:       &manager.DeviceManager{},
			deviceModelManager:  &manager.DeviceModelManager{},
			deviceStatusManager: &manager.DeviceStatusManager{},
			crdClient:           crdfake.NewSimpleClientset(),
			messageLayer:        fml,
		}
		device := &v1beta1.Device{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: "default",
				Name:      "dev2",
			},
			Spec: v1beta1.DeviceSpec{
				NodeName: "edge-node-1",
				DeviceModelRef: &corev1.LocalObjectReference{
					Name: "test-model",
				},
			},
		}
		deviceID := util.GetResourceID(device.Namespace, device.Name)
		// Simulate a corrupted cache entry.
		dc.deviceManager.Device.Store(deviceID, "not-a-device")

		assert.NotPanics(t, func() {
			dc.deviceUpdated(device)
		})

		// The corrupted entry triggers deviceAdded which sends a membership message
		// to the edge node — verify the downlink path was not silently skipped.
		assert.NotEmpty(t, fml.sent, "deviceAdded should have sent at least one message to the edge node")

		// Cache must hold the new device.
		val, ok := dc.deviceManager.Device.Load(deviceID)
		assert.True(t, ok)
		_, isDevice := val.(*v1beta1.Device)
		assert.True(t, isDevice)
	})
}

func TestRemoveTwinWithNameChanged(t *testing.T) {
	tests := []struct {
		name         string
		device       *v1beta1.Device
		deviceStatus *v1beta1.DeviceStatus
		expected     []v1beta1.Twin
	}{
		{
			name: "Remove twin with changed property name",
			device: &v1beta1.Device{
				Spec: v1beta1.DeviceSpec{
					Properties: []v1beta1.DeviceProperty{
						{
							Name: "temp",
						},
						{
							Name: "humidity",
						},
					},
				},
			},
			deviceStatus: &v1beta1.DeviceStatus{
				Status: v1beta1.DeviceStatusStatus{
					Twins: []v1beta1.Twin{
						{
							PropertyName: "temp",
							Reported: v1beta1.TwinProperty{
								Value: "25",
							},
						},
						{
							PropertyName: "pressure", // This will be removed
							Reported: v1beta1.TwinProperty{
								Value: "1000",
							},
						},
						{
							PropertyName: "humidity",
							Reported: v1beta1.TwinProperty{
								Value: "60",
							},
						},
					},
				},
			},
			expected: []v1beta1.Twin{
				{
					PropertyName: "temp",
					Reported: v1beta1.TwinProperty{
						Value: "25",
					},
				},
				{
					PropertyName: "humidity",
					Reported: v1beta1.TwinProperty{
						Value: "60",
					},
				},
			},
		},
		{
			name: "No twins to remove",
			device: &v1beta1.Device{
				Spec: v1beta1.DeviceSpec{
					Properties: []v1beta1.DeviceProperty{
						{
							Name: "temp",
						},
					},
				},
			},
			deviceStatus: &v1beta1.DeviceStatus{
				Status: v1beta1.DeviceStatusStatus{
					Twins: []v1beta1.Twin{
						{
							PropertyName: "temp",
							Reported: v1beta1.TwinProperty{
								Value: "25",
							},
						},
					},
				},
			},
			expected: []v1beta1.Twin{
				{
					PropertyName: "temp",
					Reported: v1beta1.TwinProperty{
						Value: "25",
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			removeTwinWithNameChanged(tt.deviceStatus, tt.device)
			assert.Equal(t, tt.expected, tt.deviceStatus.Status.Twins)
		})
	}
}
