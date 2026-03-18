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

package edit

import (
	"context"
	"io"
	"reflect"
	"testing"

	"github.com/agiledragon/gomonkey/v2"
	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/rest"
	"k8s.io/kubectl/pkg/cmd/util/editor"

	cfgv1alpha2 "github.com/kubeedge/api/apis/componentconfig/edgecore/v1alpha2"
	"github.com/kubeedge/api/apis/devices/v1beta1"
	"github.com/kubeedge/kubeedge/keadm/cmd/keadm/app/cmd/ctl/client"
	"github.com/kubeedge/kubeedge/keadm/cmd/keadm/app/cmd/util"
)

func TestNewEditDeviceOpts(t *testing.T) {
	opts := NewEditDeviceOpts()
	assert.NotNil(t, opts)
}

func TestEditDevice(t *testing.T) {
	tests := []struct {
		name           string
		args           []string
		nodeName       string
		device         *v1beta1.Device
		wantErr        bool
		expectedErrMsg string
		cancelled      bool
	}{
		{
			name:     "edit success",
			args:     []string{"test-device"},
			nodeName: "test-node",
			device: &v1beta1.Device{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-device",
					Namespace: "default",
				},
				Spec: v1beta1.DeviceSpec{
					NodeName: "test-node",
				},
			},
			wantErr: false,
		},
		{
			name:           "too many args",
			args:           []string{"device1", "device2"},
			wantErr:        true,
			expectedErrMsg: "too many args",
		},
		{
			name:     "device node name mismatch",
			args:     []string{"test-device"},
			nodeName: "test-node",
			device: &v1beta1.Device{
				Spec: v1beta1.DeviceSpec{
					NodeName: "other-node",
				},
			},
			wantErr: false, // it just logs an error
		},
		{
			name:     "edit cancelled",
			args:     []string{"test-device"},
			nodeName: "test-node",
			device: &v1beta1.Device{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-device",
					Namespace: "default",
				},
				Spec: v1beta1.DeviceSpec{
					NodeName: "test-node",
				},
			},
			wantErr:        true,
			expectedErrMsg: "no changes made",
			cancelled:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			patches := gomonkey.NewPatches()
			defer patches.Reset()

			patches.ApplyFunc(util.ParseEdgecoreConfig, func(path string) (*cfgv1alpha2.EdgeCoreConfig, error) {
				return &cfgv1alpha2.EdgeCoreConfig{
					Modules: &cfgv1alpha2.Modules{
						Edged: &cfgv1alpha2.Edged{
							TailoredKubeletFlag: cfgv1alpha2.TailoredKubeletFlag{
								HostnameOverride: tt.nodeName,
							},
						},
					},
				}, nil
			})

			patches.ApplyMethod(reflect.TypeOf(&client.DeviceRequest{}), "GetDevice", func(_ *client.DeviceRequest, _ context.Context) (*v1beta1.Device, error) {
				return tt.device, nil
			})

			patches.ApplyFunc(editor.NewDefaultEditor, func(envs []string) editor.Editor {
				return editor.Editor{}
			})

			patches.ApplyMethod(reflect.TypeOf(editor.Editor{}), "LaunchTempFile", func(_ editor.Editor, prefix, suffix string, r io.Reader) ([]byte, string, error) {
				if tt.cancelled {
					data, _ := io.ReadAll(r)
					return data, "fake-file", nil
				}
				// Return valid JSON that is different from the input YAML
				return []byte("{\"metadata\":{\"name\":\"test-device\",\"namespace\":\"default\"},\"spec\":{\"nodeName\":\"test-node\"}}"), "fake-file", nil
			})

			patches.ApplyMethod(reflect.TypeOf(&client.DeviceRequest{}), "UpdateDevice", func(_ *client.DeviceRequest, _ context.Context, _ *v1beta1.Device) (*rest.Result, error) {
				return nil, nil
			})

			o := NewEditDeviceOpts()
			err := o.editDevice(tt.args)
			if tt.wantErr {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedErrMsg)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
