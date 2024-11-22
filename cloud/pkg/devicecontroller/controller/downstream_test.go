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

	"github.com/kubeedge/api/apis/devices/v1beta1"
)

func TestRemoveTwinWithNameChanged(t *testing.T) {
	tests := []struct {
		name     string
		device   *v1beta1.Device
		expected []v1beta1.Twin
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
				Status: v1beta1.DeviceStatus{
					Twins: []v1beta1.Twin{
						{
							PropertyName: "temp",
							Reported: v1beta1.TwinProperty{
								Value: "25",
							},
						},
						{
							PropertyName: "pressure", // This will be be removed
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
				Status: v1beta1.DeviceStatus{
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
			removeTwinWithNameChanged(tt.device)
			assert.Equal(t, tt.expected, tt.device.Status.Twins)
		})
	}
}
