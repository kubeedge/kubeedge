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
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kubeedge/api/apis/devices/v1beta1"
	crdfake "github.com/kubeedge/api/client/clientset/versioned/fake"
	"github.com/kubeedge/kubeedge/cloud/pkg/devicecontroller/manager"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	ktesting "k8s.io/client-go/testing"
)

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

func TestGetOrCreateDeviceStatusForDeviceHandlesAlreadyExistsRace(t *testing.T) {
	device := &v1beta1.Device{
		TypeMeta: metav1.TypeMeta{
			APIVersion: v1beta1.SchemeGroupVersion.String(),
			Kind:       "Device",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "lite-node",
			Namespace: "led",
			UID:       types.UID("device-uid"),
		},
	}
	existingStatus := &v1beta1.DeviceStatus{
		TypeMeta: metav1.TypeMeta{
			APIVersion: v1beta1.SchemeGroupVersion.String(),
			Kind:       "DeviceStatus",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      device.Name,
			Namespace: device.Namespace,
		},
	}

	deviceStatuses := schema.GroupResource{Group: v1beta1.GroupName, Resource: "devicestatuses"}
	getCalls := 0
	crdClient := crdfake.NewSimpleClientset()
	crdClient.Fake.PrependReactor("get", "devicestatuses", func(action ktesting.Action) (bool, runtime.Object, error) {
		getCalls++
		if getCalls == 1 {
			return true, nil, apierrors.NewNotFound(deviceStatuses, action.(ktesting.GetAction).GetName())
		}
		return true, existingStatus.DeepCopy(), nil
	})
	crdClient.Fake.PrependReactor("create", "devicestatuses", func(action ktesting.Action) (bool, runtime.Object, error) {
		deviceStatus := action.(ktesting.CreateAction).GetObject().(*v1beta1.DeviceStatus)
		return true, nil, apierrors.NewAlreadyExists(deviceStatuses, deviceStatus.Name)
	})

	dc := &DownstreamController{
		crdClient: crdClient,
		deviceStatusManager: &manager.DeviceStatusManager{
			DeviceStatus: sync.Map{},
		},
	}

	status, err := dc.getOrCreateDeviceStatusForDevice(device)
	assert.NoError(t, err)
	assert.Equal(t, existingStatus.Name, status.Name)
	assert.Equal(t, 2, getCalls)
}
