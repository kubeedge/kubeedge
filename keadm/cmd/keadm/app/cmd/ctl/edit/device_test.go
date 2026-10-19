/*
Copyright 2026 The KubeEdge Authors.

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
	"bytes"
	"context"
	"errors"
	"io"
	"os"
	"reflect"
	"testing"

	"github.com/agiledragon/gomonkey/v2"
	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/rest"
	"k8s.io/kubectl/pkg/cmd/util/editor"

	"github.com/kubeedge/api/apis/componentconfig/edgecore/v1alpha2"
	"github.com/kubeedge/api/apis/devices/v1beta1"
	"github.com/kubeedge/kubeedge/keadm/cmd/keadm/app/cmd/common"
	"github.com/kubeedge/kubeedge/keadm/cmd/keadm/app/cmd/ctl/client"
	"github.com/kubeedge/kubeedge/keadm/cmd/keadm/app/cmd/util"
)

const (
	deviceTestNodeName = "test-node"
	testDeviceName     = "test-device"
	testTempFile       = "temp-file"
)

func setupEditDeviceTest() (*DeviceEditOptions, *v1alpha2.EdgeCoreConfig, *gomonkey.Patches) {
	opts := NewEditDeviceOpts()

	edgeCoreConfig := v1alpha2.NewDefaultEdgeCoreConfig()
	edgeCoreConfig.Modules.Edged.HostnameOverride = deviceTestNodeName

	patches := gomonkey.ApplyFunc(util.ParseEdgecoreConfig,
		func(configPath string) (*v1alpha2.EdgeCoreConfig, error) {
			return edgeCoreConfig, nil
		})

	return opts, edgeCoreConfig, patches
}

func TestNewEdgeEdit(t *testing.T) {
	cmd := NewEdgeEdit()
	assert.NotNil(t, cmd)
	assert.Equal(t, "edit", cmd.Use)
	assert.Equal(t, edgeEditShortDescription, cmd.Short)
}

func TestNewEdgeEditDevice(t *testing.T) {
	cmd := NewEdgeEditDevice()
	assert.NotNil(t, cmd)
	assert.Equal(t, "device", cmd.Use)
	assert.Equal(t, edgeEditDeviceShortDescription, cmd.Short)

	// Verify namespace flag
	flagNamespace := cmd.Flags().Lookup(common.FlagNameNamespace)
	assert.NotNil(t, flagNamespace)
	assert.Equal(t, "n", flagNamespace.Shorthand)
	assert.Equal(t, "default", flagNamespace.DefValue)
}

func TestNewEditDeviceOpts(t *testing.T) {
	opts := NewEditDeviceOpts()
	assert.NotNil(t, opts)
	assert.NotNil(t, opts.In)
	assert.NotNil(t, opts.Out)
	assert.NotNil(t, opts.ErrOut)
	assert.NotNil(t, opts.editPrinterOptions)
}

func TestEditDeviceArgumentValidation(t *testing.T) {
	opts, _, patches := setupEditDeviceTest()
	defer patches.Reset()

	// Case 1: 0 arguments passed
	err := opts.editDevice([]string{})
	assert.Error(t, err)
	assert.Equal(t, "device name is required", err.Error())

	// Case 2: >1 arguments passed
	err = opts.editDevice([]string{"dev1", "dev2"})
	assert.Error(t, err)
	assert.Equal(t, "too many args, edit one device at once", err.Error())
}

func TestEditDeviceZeroArgsDoesNotParseConfig(t *testing.T) {
	opts := NewEditDeviceOpts()

	called := false
	patches := gomonkey.ApplyFunc(util.ParseEdgecoreConfig,
		func(configPath string) (*v1alpha2.EdgeCoreConfig, error) {
			called = true
			return nil, errors.New("failed to parse config")
		})
	defer patches.Reset()

	err := opts.editDevice([]string{})
	assert.Error(t, err)
	assert.Equal(t, "device name is required", err.Error())
	assert.False(t, called, "ParseEdgecoreConfig should not be called before argument validation")
}

func TestEditDeviceErrorConfig(t *testing.T) {
	opts := NewEditDeviceOpts()
	patches := gomonkey.ApplyFunc(util.ParseEdgecoreConfig,
		func(configPath string) (*v1alpha2.EdgeCoreConfig, error) {
			return nil, errors.New("failed to parse config")
		})
	defer patches.Reset()

	err := opts.editDevice([]string{testDeviceName})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "get edge config failed")
}

func TestEditDeviceGetDeviceError(t *testing.T) {
	opts, _, patches := setupEditDeviceTest()
	defer patches.Reset()

	patches.ApplyMethod(reflect.TypeOf(&client.DeviceRequest{}), "GetDevice",
		func(_ *client.DeviceRequest, _ context.Context) (*v1beta1.Device, error) {
			return nil, errors.New("device not found")
		})

	err := opts.editDevice([]string{testDeviceName})
	assert.Error(t, err)
	assert.Equal(t, "device not found", err.Error())
}

func TestEditDeviceNodeMismatch(t *testing.T) {
	opts, _, patches := setupEditDeviceTest()
	defer patches.Reset()

	patches.ApplyMethod(reflect.TypeOf(&client.DeviceRequest{}), "GetDevice",
		func(_ *client.DeviceRequest, _ context.Context) (*v1beta1.Device, error) {
			return &v1beta1.Device{
				ObjectMeta: metav1.ObjectMeta{Name: testDeviceName, Namespace: "default"},
				Spec: v1beta1.DeviceSpec{
					NodeName: "other-node",
				},
			}, nil
		})

	err := opts.editDevice([]string{testDeviceName})
	assert.NoError(t, err)
}

func TestPrintObj(t *testing.T) {
	opts := NewEditDeviceOpts()
	device := &v1beta1.Device{
		ObjectMeta: metav1.ObjectMeta{Name: testDeviceName, Namespace: "default"},
		Spec: v1beta1.DeviceSpec{
			NodeName: deviceTestNodeName,
		},
	}

	buf := &bytes.Buffer{}
	err := opts.editPrinterOptions.PrintObj(device, buf)
	assert.NoError(t, err)
	assert.Contains(t, buf.String(), testDeviceName)
}

func TestEditDeviceSuccess(t *testing.T) {
	opts, _, patches := setupEditDeviceTest()
	defer patches.Reset()

	device := &v1beta1.Device{
		ObjectMeta: metav1.ObjectMeta{Name: testDeviceName, Namespace: "default"},
		Spec: v1beta1.DeviceSpec{
			NodeName: deviceTestNodeName,
		},
	}

	patches.ApplyMethod(reflect.TypeOf(&client.DeviceRequest{}), "GetDevice",
		func(_ *client.DeviceRequest, _ context.Context) (*v1beta1.Device, error) {
			return device, nil
		})

	modifiedYAML := []byte("apiVersion: devices.kubeedge.io/v1beta1\nkind: Device\nmetadata:\n  name: test-device\n  namespace: default\nspec:\n  nodeName: test-node\n  description: updated-device\n")

	patches.ApplyMethod(reflect.TypeOf(editor.NewDefaultEditor([]string{})), "LaunchTempFile",
		func(_ editor.Editor, _ string, _ string, _ io.Reader) ([]byte, string, error) {
			return modifiedYAML, testTempFile, nil
		})

	patches.ApplyFunc(os.Remove, func(name string) error {
		return nil
	})

	patches.ApplyMethod(reflect.TypeOf(&client.DeviceRequest{}), "UpdateDevice",
		func(_ *client.DeviceRequest, _ context.Context, _ *v1beta1.Device) (*rest.Result, error) {
			return &rest.Result{}, nil
		})

	err := opts.editDevice([]string{testDeviceName})
	assert.NoError(t, err)
}

func TestEditDeviceCancelled(t *testing.T) {
	opts, _, patches := setupEditDeviceTest()
	defer patches.Reset()

	device := &v1beta1.Device{
		ObjectMeta: metav1.ObjectMeta{Name: testDeviceName, Namespace: "default"},
		Spec: v1beta1.DeviceSpec{
			NodeName: deviceTestNodeName,
		},
	}

	patches.ApplyMethod(reflect.TypeOf(&client.DeviceRequest{}), "GetDevice",
		func(_ *client.DeviceRequest, _ context.Context) (*v1beta1.Device, error) {
			return device, nil
		})

	patches.ApplyMethod(reflect.TypeOf(editor.NewDefaultEditor([]string{})), "LaunchTempFile",
		func(_ editor.Editor, _ string, _ string, r io.Reader) ([]byte, string, error) {
			buf := new(bytes.Buffer)
			_, _ = buf.ReadFrom(r)
			return buf.Bytes(), testTempFile, nil
		})

	patches.ApplyFunc(os.Remove, func(name string) error {
		return nil
	})

	err := opts.editDevice([]string{testDeviceName})
	assert.Error(t, err)
	assert.Equal(t, "no changes made", err.Error())
}

func TestEditDeviceUnmarshalError(t *testing.T) {
	opts, _, patches := setupEditDeviceTest()
	defer patches.Reset()

	device := &v1beta1.Device{
		ObjectMeta: metav1.ObjectMeta{Name: testDeviceName, Namespace: "default"},
		Spec: v1beta1.DeviceSpec{
			NodeName: deviceTestNodeName,
		},
	}

	patches.ApplyMethod(reflect.TypeOf(&client.DeviceRequest{}), "GetDevice",
		func(_ *client.DeviceRequest, _ context.Context) (*v1beta1.Device, error) {
			return device, nil
		})

	invalidYAML := []byte("invalid_yaml: [unclosed_bracket")

	patches.ApplyMethod(reflect.TypeOf(editor.NewDefaultEditor([]string{})), "LaunchTempFile",
		func(_ editor.Editor, _ string, _ string, _ io.Reader) ([]byte, string, error) {
			return invalidYAML, testTempFile, nil
		})

	patches.ApplyFunc(os.Remove, func(name string) error {
		return nil
	})

	err := opts.editDevice([]string{testDeviceName})
	assert.Error(t, err)
}
