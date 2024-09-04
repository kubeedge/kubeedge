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

package admissioncontroller

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	admissionv1 "k8s.io/api/admission/v1"
	"k8s.io/apimachinery/pkg/runtime"

	devicesv1beta1 "github.com/kubeedge/api/apis/devices/v1beta1"
)

func TestAdmitDevice(t *testing.T) {
	assert := assert.New(t)

	testCases := []struct {
		name            string
		operation       admissionv1.Operation
		device          *devicesv1beta1.Device
		expectedAllowed bool
		expectedMessage string
	}{
		{
			name:      "Create valid device",
			operation: admissionv1.Create,
			device: &devicesv1beta1.Device{
				Spec: devicesv1beta1.DeviceSpec{
					Properties: []devicesv1beta1.DeviceProperty{
						{Name: "prop1"},
						{Name: "prop2"},
					},
				},
			},
			expectedAllowed: true,
			expectedMessage: "",
		},
		{
			name:      "Create invalid device with duplicate properties",
			operation: admissionv1.Create,
			device: &devicesv1beta1.Device{
				Spec: devicesv1beta1.DeviceSpec{
					Properties: []devicesv1beta1.DeviceProperty{
						{Name: "prop1"},
						{Name: "prop1"},
					},
				},
			},
			expectedAllowed: false,
			expectedMessage: "property names must be unique.",
		},
		{
			name:      "Delete operation",
			operation: admissionv1.Delete,
			device: &devicesv1beta1.Device{
				Spec: devicesv1beta1.DeviceSpec{
					Properties: []devicesv1beta1.DeviceProperty{
						{Name: "prop1"},
						{Name: "prop2"},
					},
				},
			},
			expectedAllowed: true,
			expectedMessage: "",
		},
		{
			name:            "Unsupported operation",
			operation:       "UnsupportedOperation",
			device:          nil,
			expectedAllowed: false,
			expectedMessage: "Unsupported webhook operation!",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			var raw []byte
			var err error
			if tc.device != nil {
				raw, err = json.Marshal(tc.device)
				assert.NoError(err)
			}

			review := admissionv1.AdmissionReview{
				Request: &admissionv1.AdmissionRequest{
					Operation: tc.operation,
					Object: runtime.RawExtension{
						Raw: raw,
					},
				},
			}

			response := admitDevice(review)

			assert.Equal(tc.expectedAllowed, response.Allowed)
			if tc.expectedMessage != "" {
				assert.Equal(tc.expectedMessage, response.Result.Message)
			} else {
				assert.Nil(response.Result)
			}
		})
	}
}

func TestValidateDevice(t *testing.T) {
	assert := assert.New(t)

	testCases := []struct {
		name            string
		device          *devicesv1beta1.Device
		expectedAllowed bool
		expectedMessage string
	}{
		{
			name: "Device with unique properties",
			device: &devicesv1beta1.Device{
				Spec: devicesv1beta1.DeviceSpec{
					Properties: []devicesv1beta1.DeviceProperty{
						{Name: "prop1"},
						{Name: "prop2"},
						{Name: "prop3"},
					},
				},
			},
			expectedAllowed: true,
			expectedMessage: "",
		},
		{
			name: "Device with duplicate properties",
			device: &devicesv1beta1.Device{
				Spec: devicesv1beta1.DeviceSpec{
					Properties: []devicesv1beta1.DeviceProperty{
						{Name: "prop1"},
						{Name: "prop2"},
						{Name: "prop1"},
					},
				},
			},
			expectedAllowed: false,
			expectedMessage: "property names must be unique.",
		},
		{
			name: "Device with no properties",
			device: &devicesv1beta1.Device{
				Spec: devicesv1beta1.DeviceSpec{
					Properties: []devicesv1beta1.DeviceProperty{},
				},
			},
			expectedAllowed: true,
			expectedMessage: "",
		},
		{
			name: "Device with one property",
			device: &devicesv1beta1.Device{
				Spec: devicesv1beta1.DeviceSpec{
					Properties: []devicesv1beta1.DeviceProperty{
						{Name: "prop1"},
					},
				},
			},
			expectedAllowed: true,
			expectedMessage: "",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			response := &admissionv1.AdmissionResponse{
				Allowed: true,
			}

			msg := validateDevice(tc.device, response)

			assert.Equal(tc.expectedAllowed, response.Allowed)
			assert.Equal(tc.expectedMessage, msg)
		})
	}
}
