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

func TestNewUpstreamController(t *testing.T) {
	assert := assert.New(t)

	dc := &DownstreamController{}
	uc, err := NewUpstreamController(dc)
	assert.NoError(err)
	assert.NotNil(uc)

	assert.NotNil(uc.messageLayer)
	assert.NotNil(uc.dc)
	assert.Equal(dc, uc.dc)

	// Channels are not initialized (they should be initialized in Start())
	assert.Nil(uc.deviceTwinsChan)
	assert.Nil(uc.deviceStatesChan)
}

func TestFindOrCreateTwinByName(t *testing.T) {
	assert := assert.New(t)

	tests := []struct {
		name       string
		twinName   string
		properties []v1beta1.DeviceProperty
		status     *DeviceStatus
		expected   *v1beta1.Twin
	}{
		{
			name:     "finding existing twin",
			twinName: "temperature",
			properties: []v1beta1.DeviceProperty{
				{
					Name: "temperature",
				},
			},
			status: &DeviceStatus{
				Status: v1beta1.DeviceStatus{
					Twins: []v1beta1.Twin{
						{
							PropertyName: "temperature",
							Reported: v1beta1.TwinProperty{
								Value: "25",
							},
						},
					},
				},
			},
			expected: &v1beta1.Twin{
				PropertyName: "temperature",
				Reported: v1beta1.TwinProperty{
					Value: "25",
				},
			},
		},
		{
			name:     "creating new twin",
			twinName: "humidity",
			properties: []v1beta1.DeviceProperty{
				{
					Name: "humidity",
				},
			},
			status: &DeviceStatus{
				Status: v1beta1.DeviceStatus{
					Twins: []v1beta1.Twin{},
				},
			},
			expected: &v1beta1.Twin{
				PropertyName: "humidity",
			},
		},
		{
			name:     "property not found",
			twinName: "nonexistent",
			properties: []v1beta1.DeviceProperty{
				{
					Name: "temperature",
				},
			},
			status: &DeviceStatus{
				Status: v1beta1.DeviceStatus{
					Twins: []v1beta1.Twin{},
				},
			},
			expected: nil,
		},
		{
			name:     "multiple properties",
			twinName: "temperature",
			properties: []v1beta1.DeviceProperty{
				{
					Name: "humidity",
				},
				{
					Name: "temperature",
				},
			},
			status: &DeviceStatus{
				Status: v1beta1.DeviceStatus{
					Twins: []v1beta1.Twin{
						{
							PropertyName: "humidity",
							Reported: v1beta1.TwinProperty{
								Value: "60",
							},
						},
					},
				},
			},
			expected: &v1beta1.Twin{
				PropertyName: "temperature",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := findOrCreateTwinByName(tt.twinName, tt.properties, tt.status)
			if tt.expected == nil {
				assert.Nil(result)
			} else {
				assert.Equal(tt.expected.PropertyName, result.PropertyName)
				if tt.expected.Reported.Value != "" {
					assert.Equal(tt.expected.Reported, result.Reported)
				}
				// Verify twin was added to DeviceStatus if created
				if len(tt.status.Status.Twins) > 0 {
					found := false
					for _, twin := range tt.status.Status.Twins {
						if twin.PropertyName == tt.twinName {
							found = true
							break
						}
					}
					assert.True(found)
				}
			}
		})
	}
}

func TestFindTwinByName(t *testing.T) {
	tests := []struct {
		name         string
		twinName     string
		deviceStatus *DeviceStatus
		expected     *v1beta1.Twin
	}{
		{
			name:     "twin exists",
			twinName: "temperature",
			deviceStatus: &DeviceStatus{
				Status: v1beta1.DeviceStatus{
					Twins: []v1beta1.Twin{
						{
							PropertyName: "temperature",
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
			},
			expected: &v1beta1.Twin{
				PropertyName: "temperature",
				Reported: v1beta1.TwinProperty{
					Value: "25",
				},
			},
		},
		{
			name:     "twin doesn't exist",
			twinName: "pressure",
			deviceStatus: &DeviceStatus{
				Status: v1beta1.DeviceStatus{
					Twins: []v1beta1.Twin{
						{
							PropertyName: "temperature",
							Reported: v1beta1.TwinProperty{
								Value: "25",
							},
						},
					},
				},
			},
			expected: nil,
		},
		{
			name:     "device status is nil",
			twinName: "temperature",
			deviceStatus: &DeviceStatus{
				Status: v1beta1.DeviceStatus{},
			},
			expected: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := findTwinByName(tt.twinName, tt.deviceStatus)
			if tt.expected == nil {
				assert.Nil(t, result)
			} else {
				assert.Equal(t, tt.expected.PropertyName, result.PropertyName)
				assert.Equal(t, tt.expected.Reported, result.Reported)
			}
		})
	}
}
