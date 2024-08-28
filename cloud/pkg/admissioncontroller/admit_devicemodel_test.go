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

func TestAdmitDeviceModel(t *testing.T) {
	assert := assert.New(t)

	testCases := []struct {
		name            string
		operation       admissionv1.Operation
		deviceModel     *devicesv1beta1.DeviceModel
		expectedAllowed bool
		expectedMessage string
	}{
		{
			name:      "Create valid device model",
			operation: admissionv1.Create,
			deviceModel: &devicesv1beta1.DeviceModel{
				Spec: devicesv1beta1.DeviceModelSpec{
					Properties: []devicesv1beta1.ModelProperty{
						{Name: "prop1"},
						{Name: "prop2"},
					},
				},
			},
			expectedAllowed: true,
			expectedMessage: "",
		},
		{
			name:      "Create invalid device model with duplicate properties",
			operation: admissionv1.Create,
			deviceModel: &devicesv1beta1.DeviceModel{
				Spec: devicesv1beta1.DeviceModelSpec{
					Properties: []devicesv1beta1.ModelProperty{
						{Name: "prop1"},
						{Name: "prop1"},
					},
				},
			},
			expectedAllowed: false,
			expectedMessage: "property names must be unique.",
		},
		{
			name:            "Delete operation",
			operation:       admissionv1.Delete,
			deviceModel:     nil,
			expectedAllowed: true,
			expectedMessage: "",
		},
		{
			name:            "Unsupported operation",
			operation:       "UnsupportedOperation",
			deviceModel:     nil,
			expectedAllowed: false,
			expectedMessage: "Unsupported webhook operation!",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			var raw []byte
			var err error
			if tc.deviceModel != nil {
				raw, err = json.Marshal(tc.deviceModel)
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

			response := admitDeviceModel(review)

			assert.Equal(tc.expectedAllowed, response.Allowed)
			if tc.expectedMessage != "" {
				assert.Equal(tc.expectedMessage, response.Result.Message)
			} else {
				assert.Nil(response.Result)
			}
		})
	}
}

func TestValidateDeviceModel(t *testing.T) {
	assert := assert.New(t)

	testCases := []struct {
		name            string
		deviceModel     *devicesv1beta1.DeviceModel
		expectedAllowed bool
		expectedMessage string
	}{
		{
			name: "Device model with unique properties",
			deviceModel: &devicesv1beta1.DeviceModel{
				Spec: devicesv1beta1.DeviceModelSpec{
					Properties: []devicesv1beta1.ModelProperty{
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
			name: "Device model with duplicate properties",
			deviceModel: &devicesv1beta1.DeviceModel{
				Spec: devicesv1beta1.DeviceModelSpec{
					Properties: []devicesv1beta1.ModelProperty{
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
			name: "Device model with no properties",
			deviceModel: &devicesv1beta1.DeviceModel{
				Spec: devicesv1beta1.DeviceModelSpec{
					Properties: []devicesv1beta1.ModelProperty{},
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

			msg := validateDeviceModel(tc.deviceModel, response)

			assert.Equal(tc.expectedAllowed, response.Allowed)
			assert.Equal(tc.expectedMessage, msg)
		})
	}
}
