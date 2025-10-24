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

	"github.com/kubeedge/api/apis/devices/v1beta1"
	"github.com/stretchr/testify/assert"
)

func TestNewUpstreamController(t *testing.T) {
	a := assert.New(t)

	dc := &DownstreamController{}
	uc, err := NewUpstreamController(dc)

	a.NoError(err)
	a.NotNil(uc)
	a.NotNil(uc.messageLayer)
	a.NotNil(uc.dc)
	a.Equal(dc, uc.dc)

	// Channels are not initialized (should be initialized in Start()).
	a.Nil(uc.deviceTwinsChan)
	a.Nil(uc.deviceStatesChan)
}

func TestFindOrCreateTwinByName(t *testing.T) {
	type twinCase struct {
		name       string
		twinName   string
		properties []string
		initial    []twinData
		expected   *twinData
	}

	cases := []twinCase{
		{
			name:       "finding existing twin",
			twinName:   "temperature",
			properties: []string{"temperature"},
			initial:    []twinData{{"temperature", "25"}},
			expected:   &twinData{"temperature", "25"},
		},
		{
			name:       "creating new twin",
			twinName:   "humidity",
			properties: []string{"humidity"},
			initial:    nil,
			expected:   &twinData{"humidity", ""},
		},
		{
			name:       "property not found",
			twinName:   "nonexistent",
			properties: []string{"temperature"},
			initial:    nil,
			expected:   nil,
		},
		{
			name:       "multiple properties",
			twinName:   "temperature",
			properties: []string{"humidity", "temperature"},
			initial:    []twinData{{"humidity", "60"}},
			expected:   &twinData{"temperature", ""},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			status := &DeviceStatus{
				Status: v1beta1.DeviceStatus{Twins: buildTwins(tc.initial)},
			}
			props := buildProperties(tc.properties)

			result := findOrCreateTwinByName(tc.twinName, props, status)
			if tc.expected == nil {
				assert.Nil(t, result)
				return
			}

			assert.Equal(t, tc.expected.name, result.PropertyName)
			if tc.expected.value != "" {
				assert.Equal(t, tc.expected.value, result.Reported.Value)
			}

			// Verify twin was added if it was supposed to be created.
			found := false
			for _, twin := range status.Status.Twins {
				if twin.PropertyName == tc.twinName {
					found = true
					break
				}
			}
			assert.True(t, found)
		})
	}
}

func TestFindTwinByName(t *testing.T) {
	type twinCase struct {
		name     string
		twinName string
		initial  []twinData
		expected *twinData
	}

	cases := []twinCase{
		{
			name:     "twin exists",
			twinName: "temperature",
			initial:  []twinData{{"temperature", "25"}, {"humidity", "60"}},
			expected: &twinData{"temperature", "25"},
		},
		{
			name:     "twin doesn't exist",
			twinName: "pressure",
			initial:  []twinData{{"temperature", "25"}},
			expected: nil,
		},
		{
			name:     "device status empty",
			twinName: "temperature",
			initial:  nil,
			expected: nil,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			status := &DeviceStatus{
				Status: v1beta1.DeviceStatus{Twins: buildTwins(tc.initial)},
			}

			result := findTwinByName(tc.twinName, status)
			if tc.expected == nil {
				assert.Nil(t, result)
				return
			}

			assert.Equal(t, tc.expected.name, result.PropertyName)
			assert.Equal(t, tc.expected.value, result.Reported.Value)
		})
	}
}

// ---------- helpers to reduce boilerplate ----------

type twinData struct {
	name  string
	value string
}

func buildProperties(names []string) []v1beta1.DeviceProperty {
	props := make([]v1beta1.DeviceProperty, len(names))
	for i, n := range names {
		props[i] = v1beta1.DeviceProperty{Name: n}
	}
	return props
}

func buildTwins(data []twinData) []v1beta1.Twin {
	twins := make([]v1beta1.Twin, len(data))
	for i, d := range data {
		twins[i] = v1beta1.Twin{
			PropertyName: d.name,
			Reported:     v1beta1.TwinProperty{Value: d.value},
		}
	}
	return twins
}
