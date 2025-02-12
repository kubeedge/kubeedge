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

	"github.com/stretchr/testify/assert"
	"google.golang.org/protobuf/types/known/anypb"
	"google.golang.org/protobuf/types/known/wrapperspb"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/kubeedge/api/apis/devices/v1beta1"
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
			isValidate := ValidateTwinKey(test.key)
			if test.want != isValidate {
				t.Errorf("ValidateTwinKey Case failed: wanted %v and got %v", test.want, isValidate)
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
			isValidate := ValidateTwinValue(test.key)
			if test.want != isValidate {
				t.Errorf("ValidateTwinValue Case failed: wanted %v and got %v", test.want, isValidate)
			}
		})
	}
}
func TestDataToAny(t *testing.T) {
	cases := []struct {
		name    string
		input   interface{}
		want    *anypb.Any
		wantErr bool
	}{
		{
			name:  "string value",
			input: "test",
			want: func() *anypb.Any {
				a, _ := anypb.New(wrapperspb.String("test"))
				return a
			}(),
			wantErr: false,
		},
		{
			name:  "int value",
			input: int(42),
			want: func() *anypb.Any {
				a, _ := anypb.New(wrapperspb.Int32(42))
				return a
			}(),
			wantErr: false,
		},
		{
			name:  "int8 value",
			input: int8(8),
			want: func() *anypb.Any {
				a, _ := anypb.New(wrapperspb.Int32(8))
				return a
			}(),
			wantErr: false,
		},
		{
			name:  "int16 value",
			input: int16(16),
			want: func() *anypb.Any {
				a, _ := anypb.New(wrapperspb.Int32(16))
				return a
			}(),
			wantErr: false,
		},
		{
			name:  "int32 value",
			input: int32(32),
			want: func() *anypb.Any {
				a, _ := anypb.New(wrapperspb.Int32(32))
				return a
			}(),
			wantErr: false,
		},
		{
			name:  "int64 value",
			input: int64(64),
			want: func() *anypb.Any {
				a, _ := anypb.New(wrapperspb.Int64(64))
				return a
			}(),
			wantErr: false,
		},
		{
			name:  "float32 value",
			input: float32(3.14),
			want: func() *anypb.Any {
				a, _ := anypb.New(wrapperspb.Float(3.14))
				return a
			}(),
			wantErr: false,
		},
		{
			name:  "float64 value",
			input: float64(6.28),
			want: func() *anypb.Any {
				a, _ := anypb.New(wrapperspb.Float(float32(6.28)))
				return a
			}(),
			wantErr: false,
		},
		{
			name:  "bool value",
			input: true,
			want: func() *anypb.Any {
				a, _ := anypb.New(wrapperspb.Bool(true))
				return a
			}(),
			wantErr: false,
		},
		{
			name:    "unsupported type",
			input:   make(chan int),
			want:    nil,
			wantErr: true,
		},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			got, err := dataToAny(tt.input)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
			assert.Equal(t, tt.want.TypeUrl, got.TypeUrl)
		})
	}
}

func TestConvertDevice(t *testing.T) {
	cases := []struct {
		name    string
		device  *v1beta1.Device
		wantErr bool
		errMsg  string
	}{
		{
			name: "basic device conversion",
			device: &v1beta1.Device{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-device",
					Namespace: "default",
				},
				Spec: v1beta1.DeviceSpec{
					DeviceModelRef: &v1.LocalObjectReference{
						Name: "test-model",
					},
					Protocol: v1beta1.ProtocolConfig{
						ProtocolName: "mqtt",
						ConfigData: &v1beta1.CustomizedValue{
							Data: map[string]interface{}{
								"interval": "10",
							},
						},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "device with properties",
			device: &v1beta1.Device{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-device-props",
					Namespace: "default",
				},
				Spec: v1beta1.DeviceSpec{
					DeviceModelRef: &v1.LocalObjectReference{
						Name: "test-model",
					},
					Properties: []v1beta1.DeviceProperty{
						{
							Name: "temperature",
							Visitors: v1beta1.VisitorConfig{
								ProtocolName: "mqtt",
								ConfigData: &v1beta1.CustomizedValue{
									Data: map[string]interface{}{
										"topic": "temp",
									},
								},
							},
						},
					},
				},
			},
			wantErr: false,
		},
		{
			name:    "nil device",
			device:  nil,
			wantErr: true,
			errMsg:  "device cannot be nil",
		},
		{
			name: "empty ConfigData",
			device: &v1beta1.Device{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-device-empty-config",
				},
				Spec: v1beta1.DeviceSpec{
					Protocol: v1beta1.ProtocolConfig{
						ConfigData: &v1beta1.CustomizedValue{},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "nil DeviceModelRef",
			device: &v1beta1.Device{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-device-no-model",
				},
				Spec: v1beta1.DeviceSpec{},
			},
			wantErr: false,
		},
		{
			name: "device with invalid configData type",
			device: &v1beta1.Device{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-device-invalid-config",
				},
				Spec: v1beta1.DeviceSpec{
					Protocol: v1beta1.ProtocolConfig{
						ConfigData: &v1beta1.CustomizedValue{
							Data: map[string]interface{}{
								"invalid": make(chan int),
							},
						},
					},
				},
			},
			wantErr: true,
			errMsg:  "json: unsupported type: chan int",
		},
		{
			name: "device with invalid property visitor config",
			device: &v1beta1.Device{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-device-invalid-visitor",
				},
				Spec: v1beta1.DeviceSpec{
					Properties: []v1beta1.DeviceProperty{
						{
							Visitors: v1beta1.VisitorConfig{
								ConfigData: &v1beta1.CustomizedValue{
									Data: map[string]interface{}{
										"invalid": make(chan int),
									},
								},
							},
						},
					},
				},
			},
			wantErr: true,
			errMsg:  "json: unsupported type: chan int",
		},
		{
			name: "device with nil property",
			device: &v1beta1.Device{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-device-nil-property",
				},
				Spec: v1beta1.DeviceSpec{
					Properties: []v1beta1.DeviceProperty{{}},
				},
			},
			wantErr: false,
		},
		{
			name: "device with multiple properties",
			device: &v1beta1.Device{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-device-multiple-props",
				},
				Spec: v1beta1.DeviceSpec{
					Properties: []v1beta1.DeviceProperty{
						{
							Name: "temp",
							Visitors: v1beta1.VisitorConfig{
								ConfigData: &v1beta1.CustomizedValue{
									Data: map[string]interface{}{
										"topic": "temperature",
									},
								},
							},
						},
						{
							Name: "humidity",
							Visitors: v1beta1.VisitorConfig{
								ConfigData: &v1beta1.CustomizedValue{
									Data: map[string]interface{}{
										"topic": "humidity",
									},
								},
							},
						},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "protocol config data conversion error",
			device: &v1beta1.Device{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-device-protocol-error",
				},
				Spec: v1beta1.DeviceSpec{
					Protocol: v1beta1.ProtocolConfig{
						ConfigData: &v1beta1.CustomizedValue{
							Data: map[string]interface{}{
								"invalid": struct{}{}, // This is JSON marshalable but will fail conversion to Any
							},
						},
					},
				},
			},
			wantErr: true,
			errMsg:  "failed to convert protocol config data",
		},
		{
			name: "property conversion error",
			device: &v1beta1.Device{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-device-property-error",
				},
				Spec: v1beta1.DeviceSpec{
					Properties: []v1beta1.DeviceProperty{
						{
							Visitors: v1beta1.VisitorConfig{
								ConfigData: &v1beta1.CustomizedValue{
									Data: map[string]interface{}{
										"bad": struct{}{}, // This is JSON marshalable but will fail conversion to Any
									},
								},
							},
						},
					},
				},
			},
			wantErr: true,
			errMsg:  "failed to convert property",
		},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ConvertDevice(tt.device)
			if tt.wantErr {
				assert.Error(t, err)
				if tt.errMsg != "" {
					assert.Contains(t, err.Error(), tt.errMsg)
				}
				return
			}

			assert.NoError(t, err)
			assert.NotNil(t, got)
			assert.Equal(t, tt.device.Name, got.Name)
			assert.Equal(t, tt.device.Namespace, got.Namespace)

			if tt.device.Spec.DeviceModelRef != nil {
				assert.Equal(t, tt.device.Spec.DeviceModelRef.Name, got.Spec.DeviceModelReference)
			} else {
				assert.Empty(t, got.Spec.DeviceModelReference)
			}

			// Verify properties were converted correctly
			if len(tt.device.Spec.Properties) > 0 {
				assert.Equal(t, len(tt.device.Spec.Properties), len(got.Spec.Properties))
				for i, prop := range tt.device.Spec.Properties {
					assert.Equal(t, prop.Name, got.Spec.Properties[i].Name)
				}
			}
		})
	}
}
func TestConvertDeviceModel(t *testing.T) {
	cases := []struct {
		name    string
		model   *v1beta1.DeviceModel
		wantErr bool
	}{
		{
			name: "basic model conversion",
			model: &v1beta1.DeviceModel{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-model",
					Namespace: "default",
				},
			},
			wantErr: false,
		},
		{
			name: "model with complex data",
			model: &v1beta1.DeviceModel{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-model-complex",
					Namespace: "default",
				},
				TypeMeta: metav1.TypeMeta{
					Kind:       "DeviceModel",
					APIVersion: "devices.kubeedge.io/v1beta1",
				},
			},
			wantErr: false,
		},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ConvertDeviceModel(tt.model)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
			assert.Equal(t, tt.model.Name, got.Name)
			assert.Equal(t, tt.model.Namespace, got.Namespace)
		})
	}
}
func TestConvertDeviceProperty(t *testing.T) {
	invalidValue := func() {} // This will fail JSON marshaling
	type CustomValue struct {
		Value interface{} `json:"value"`
	}

	cases := []struct {
		name    string
		prop    *v1beta1.DeviceProperty
		wantErr bool
		errMsg  string
	}{
		{
			name:    "nil property",
			prop:    nil,
			wantErr: true,
			errMsg:  "property cannot be nil",
		},
		{
			name: "marshal error case",
			prop: &v1beta1.DeviceProperty{
				Name: "test-prop",
				Visitors: v1beta1.VisitorConfig{
					ConfigData: &v1beta1.CustomizedValue{
						Data: map[string]interface{}{
							"test": invalidValue,
						},
					},
				},
			},
			wantErr: true,
			errMsg:  "failed to marshal property",
		},
		{
			name: "visitor config data conversion error",
			prop: &v1beta1.DeviceProperty{
				Name: "test-prop",
				Visitors: v1beta1.VisitorConfig{
					ConfigData: &v1beta1.CustomizedValue{
						Data: map[string]interface{}{
							"bad": map[string]interface{}{}, // Empty map will marshal but fail conversion
						},
					},
				},
			},
			wantErr: true,
			errMsg:  "failed to convert visitor config data",
		},
		{
			name: "successful conversion",
			prop: &v1beta1.DeviceProperty{
				Name: "test-prop",
				Visitors: v1beta1.VisitorConfig{
					ConfigData: &v1beta1.CustomizedValue{
						Data: map[string]interface{}{
							"validKey": "validValue",
						},
					},
				},
			},
			wantErr: false,
		},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			got, err := convertDeviceProperty(tt.prop)
			if tt.wantErr {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
				return
			}
			assert.NoError(t, err)
			assert.NotNil(t, got)
			assert.Equal(t, tt.prop.Name, got.Name)
		})
	}
}
