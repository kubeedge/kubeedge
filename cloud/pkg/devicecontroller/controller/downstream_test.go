/* Copyright 2024 The KubeEdge Authors.
Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing,
software distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controller

import (
	"testing"

	"github.com/kubeedge/api/apis/devices/v1beta1"
	"github.com/stretchr/testify/assert"
)

func TestRemoveTwinWithNameChanged(t *testing.T) {
	tests := []struct {
		name     string
		device   *v1beta1.Device
		expected []v1beta1.Twin
	}{
		{
			name: "Remove twin with changed property name",
			device: buildDevice(
				[]string{"temp", "humidity"},
				[]twinData{
					{"temp", "25"},
					{"pressure", "1000"}, // should be removed
					{"humidity", "60"},
				},
			),
			expected: buildTwins(
				[]twinData{
					{"temp", "25"},
					{"humidity", "60"},
				},
			),
		},
		{
			name: "No twins to remove",
			device: buildDevice(
				[]string{"temp"},
				[]twinData{
					{"temp", "25"},
				},
			),
			expected: buildTwins(
				[]twinData{
					{"temp", "25"},
				},
			),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			removeTwinWithNameChanged(tt.device)
			assert.Equal(t, tt.expected, tt.device.Status.Twins)
		})
	}
}

// 辅助类型与构造函数，减少重复
type twinData struct {
	name  string
	value string
}

func buildDevice(props []string, twins []twinData) *v1beta1.Device {
	properties := make([]v1beta1.DeviceProperty, len(props))
	for i, p := range props {
		properties[i] = v1beta1.DeviceProperty{Name: p}
	}
	return &v1beta1.Device{
		Spec: v1beta1.DeviceSpec{
			Properties: properties,
		},
		Status: v1beta1.DeviceStatus{
			Twins: buildTwins(twins),
		},
	}
}

func buildTwins(data []twinData) []v1beta1.Twin {
	twins := make([]v1beta1.Twin, len(data))
	for i, d := range data {
		twins[i] = v1beta1.Twin{
			PropertyName: d.name,
			Reported: v1beta1.TwinProperty{
				Value: d.value,
			},
		}
	}
	return twins
}
