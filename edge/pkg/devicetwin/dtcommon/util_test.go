/*
Copyright 2018 The KubeEdge Authors.

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

package dtcommon

import (
	"errors"
	"testing"

	"k8s.io/api/core/v1"

	"github.com/kubeedge/kubeedge/pkg/apis/devices/v1beta1"
)

// TestValidateValue is function to test ValidateValue
func TestValidateValue(t *testing.T) {
	cases := []struct {
		// name is name of the testcase
		name string
		// valueType is value type of testcase, first parameter to ValidateValue function
		valueType string
		// value is value in the test case, second parameter to ValidateValue function
		value string
		// wantErr is expected error in the test case, returned by ValidateValue function
		wantErr error
	}{{
		// valuetype nil success
		name:      "ValidateValueNilSuccessCase",
		valueType: "",
		value:     "test",
		wantErr:   nil,
	}, {
		// valuetype nil success
		name:      "ValidateValueStringSuccessCase",
		valueType: "string",
		value:     "test",
		wantErr:   nil,
	}, {
		// int error
		name:      "ValidateValueIntErrorCase",
		valueType: "int",
		value:     "test",
		wantErr:   errors.New("the value is not int or integer"),
	}, {
		// float error
		name:      "ValidateValueFloatErrorCase",
		valueType: "float",
		value:     "test",
		wantErr:   errors.New("the value is not float"),
	}, {
		// bool error
		name:      "ValidateValueBoolErrorCase",
		valueType: "boolean",
		value:     "test",
		wantErr:   errors.New("the bool value must be true or false"),
	}, {
		// deleted
		name:      "ValidateValueDeletedSuccessCase",
		valueType: TypeDeleted,
		value:     "test",
		wantErr:   nil,
	}, {
		// not supported
		name:      "ValidateValueNotSupportedErrorCase",
		valueType: "test",
		value:     "test",
		wantErr:   errors.New("the value type is not allowed"),
	}, {
		// int success
		name:      "ValidateValueIntSuccessCase",
		valueType: "int",
		value:     "10",
		wantErr:   nil,
	}, {
		// float success
		name:      "ValidateValueFloatSuccessCase",
		valueType: "float",
		value:     "10.10",
		wantErr:   nil,
	}, {
		// bool success true
		name:      "ValidateValueBoolTrueSuccessCase",
		valueType: "boolean",
		value:     "true",
		wantErr:   nil,
	}, {
		// bool success false
		name:      "ValidateValueBoolFalseSuccessCase",
		valueType: "boolean",
		value:     "false",
		wantErr:   nil,
	},
	}

	// run the test cases
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			err := ValidateValue(test.valueType, test.value)
			if (err == nil && err != test.wantErr) || (err != nil && err.Error() != test.wantErr.Error()) {
				t.Errorf("TestValidateValue Case failed: wanted %v and got %v", test.wantErr, err)
			}
		})
	}
}

// TestValidateTwinKey is function to test ValidateTwinKey
func TestValidateTwinKey(t *testing.T) {
	cases := []struct {
		// name is name of the testcase
		name string
		// key is key to be validated, parameter to ValidateTwinKey function
		key string
		// want is expected boolean in test case, returned by ValidateTwinKey function
		want bool
	}{{
		// Failure case
		name: "ValidateTwinKeyFailCase",
		key:  "test^",
		want: false,
	}, {
		// Success case
		name: "ValidateTwinKeySuccessCase",
		key:  "test123",
		want: true,
	},
	}

	// run the test cases
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			bool := ValidateTwinKey(test.key)
			if test.want != bool {
				t.Errorf("ValidateTwinKey Case failed: wanted %v and got %v", test.want, bool)
			}
		})
	}
}

// TestValidateTwinValue is function to test ValidateTwinValue
func TestValidateTwinValue(t *testing.T) {
	cases := []struct {
		// name is name of the testcase
		name string
		// key is key to be validated, parameter to ValidateTwinKey function
		key string
		// want is expected boolean in test case, returned by ValidateTwinKey function
		want bool
	}{{
		// Failure case
		name: "ValidateTwinValueFailCase",
		key:  "test^",
		want: false,
	}, {
		// Success case
		name: "ValidateTwinValueSuccessCase",
		key:  "test123",
		want: true,
	},
	}

	// run the test cases
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			bool := ValidateTwinValue(test.key)
			if test.want != bool {
				t.Errorf("ValidateTwinValue Case failed: wanted %v and got %v", test.want, bool)
			}
		})
	}
}

// TestConvertDevice is function to test ConvertDevice
func TestConvertDevice(t *testing.T) {
	configTestMap := make(map[string]interface{})
	configTestMap["key1"] = "value1"
	configTestMap["key2"] = 2
	configTestMap["key3"] = true

	cases := []struct {
		// name is name of the testcase
		name string
		// device is device of the testcase
		device *v1beta1.Device
		// wantErr is expected error in the test case, returned by ValidateValue function
		wantErr error
	}{
		{
			name: "ConvertDeviceModelSuccessCase",
			device: &v1beta1.Device{
				Spec: v1beta1.DeviceSpec{
					DeviceModelRef: &v1.LocalObjectReference{
						Name: "test_model",
					},
					Protocol: v1beta1.ProtocolConfig{
						ProtocolName: "test_custiomized_protocol_name",
						ConfigData: &v1beta1.CustomizedValue{
							Data: configTestMap,
						},
					},
					Properties: []v1beta1.DeviceProperty{
						{
							Name: "test_visitor_name1",
							Visitors: v1beta1.VisitorConfig{
								ProtocolName: "test_custiomized_protocol_name",
								ConfigData: &v1beta1.CustomizedValue{
									Data: configTestMap,
								},
							},
						},
						{
							Name: "test_visitor_name2",
							Visitors: v1beta1.VisitorConfig{
								ProtocolName: "test_custiomized_protocol_name",
								ConfigData: &v1beta1.CustomizedValue{
									Data: configTestMap,
								},
							},
						},
					},
				},
				Status: v1beta1.DeviceStatus{},
			},
			wantErr: nil,
		},
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			_, err := ConvertDevice(test.device)
			if (err == nil && err != test.wantErr) || (err != nil && err.Error() != test.wantErr.Error()) {
				t.Errorf("%v failed: %v", test.name, err)
			}
		})
	}
}

// TestConvertDeviceModel is function to test ConvertDeviceModel
func TestConvertDeviceModel(t *testing.T) {
	cases := []struct {
		// name is name of the testcase
		name string
		// model is model of the testcase
		model *v1beta1.DeviceModel
		// wantErr is expected error in the test case, returned by ValidateValue function
		wantErr error
	}{
		{
			name: "ConvertDeviceModelSuccessCase",
			model: &v1beta1.DeviceModel{
				Spec: v1beta1.DeviceModelSpec{
					Protocol: "test_model",
				},
			},
		},
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			_, err := ConvertDeviceModel(test.model)
			if (err == nil && err != test.wantErr) || (err != nil && err.Error() != test.wantErr.Error()) {
				t.Errorf("%v failed: %v", test.name, err)
			}
		})
	}
}

// TestdataToAny is function to test DataToAny
func TestDataToAny(t *testing.T) {
	cases := []struct {
		// name is name of the testcase
		name string
		// key is key to be converted, parameter to dataToAny function
		key interface{}
		// wantErr is expected error in the test case, returned by ValidateValue function
		wantErr error
	}{{
		// Success case
		name:    "DataToAnyStringCase",
		key:     "test_string",
		wantErr: nil,
	}, {
		// Success case
		name:    "DataToAnyInt8Case",
		key:     int8(1),
		wantErr: nil,
	}, {
		// Success case
		name:    "DataToAnyInt16Case",
		key:     int16(1),
		wantErr: nil,
	}, {
		// Success case
		name:    "DataToAnyInt32Case",
		key:     int32(1),
		wantErr: nil,
	}, {
		// Success case
		name:    "DataToAnyInt64Case",
		key:     int64(1),
		wantErr: nil,
	}, {
		// Success case
		name:    "DataToAnyIntCase",
		key:     int(1),
		wantErr: nil,
	}, {
		// Success case
		name:    "DataToAnyFloat64Case",
		key:     1.23,
		wantErr: nil,
	}, {
		// Success case
		name:    "DataToAnyFloat32Case",
		key:     float32(1.23),
		wantErr: nil,
	}, {
		// Success case
		name:    "DataToAnyIntCase",
		key:     1,
		wantErr: nil,
	}, {
		// Success case
		name:    "DataToAnyBoolCase",
		key:     true,
		wantErr: nil,
	}, {
		//  Failure case
		name:    "DataToAnyFailureCase",
		key:     []byte{},
		wantErr: errors.New("[]uint8 does not support converting to any"),
	},
	}

	// run the test cases
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			_, err := dataToAny(test.key)
			if (err == nil && err != test.wantErr) || (err != nil && err.Error() != test.wantErr.Error()) {
				t.Errorf("%v Case failed: %v", test.name, err)
			}
		})
	}
}
