/*
Copyright 2025 The KubeEdge Authors.

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

package get

import (
	"context"
	"errors"
	"io"
	"os"
	"reflect"
	"testing"

	"github.com/agiledragon/gomonkey/v2"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/kubectl/pkg/cmd/get"

	"github.com/kubeedge/api/apis/componentconfig/edgecore/v1alpha2"
	"github.com/kubeedge/api/apis/devices/v1beta1"
	"github.com/kubeedge/kubeedge/keadm/cmd/keadm/app/cmd/common"
	"github.com/kubeedge/kubeedge/keadm/cmd/keadm/app/cmd/ctl/client"
	ctlcommon "github.com/kubeedge/kubeedge/keadm/cmd/keadm/app/cmd/ctl/common"
	"github.com/kubeedge/kubeedge/keadm/cmd/keadm/app/cmd/util"
)

const (
	deviceDefaultNamespace = "default"
	deviceTestNodeName     = "test-node"
	testDeviceName         = "test-device"
)

// setupDeviceTest creates and returns common test setup components
func setupDeviceTest() (*DeviceGetOptions, *v1alpha2.EdgeCoreConfig, *gomonkey.Patches) {
	deviceGetOptions := NewDeviceGetOpts()

	edgeCoreConfig := v1alpha2.NewDefaultEdgeCoreConfig()
	edgeCoreConfig.Modules.Edged.HostnameOverride = deviceTestNodeName

	patches := gomonkey.ApplyFunc(util.ParseEdgecoreConfig,
		func(configPath string) (*v1alpha2.EdgeCoreConfig, error) {
			return edgeCoreConfig, nil
		})

	return deviceGetOptions, edgeCoreConfig, patches
}

func TestNewEdgeDeviceGet(t *testing.T) {
	assert := assert.New(t)
	cmd := NewEdgeDeviceGet()

	assert.NotNil(cmd)
	assert.Equal("device", cmd.Use)
	assert.Equal(edgeDeviceGetShortDescription, cmd.Short)
	assert.Equal(edgeDeviceGetShortDescription, cmd.Long)

	assert.NotNil(cmd.RunE)

	assert.Equal(cmd.Flags().Lookup(common.FlagNameNamespace).Name, "namespace")
	assert.Equal(cmd.Flags().Lookup(common.FlagNameAllNamespaces).Name, "all-namespaces")
	assert.Equal(cmd.Flags().Lookup(common.FlagNameLabelSelector).Name, "selector")
	assert.Equal(cmd.Flags().Lookup(common.FlagNameOutput).Name, "output")
}

func TestNewDeviceGetOpts(t *testing.T) {
	assert := assert.New(t)

	deviceGetOptions := NewDeviceGetOpts()
	assert.NotNil(deviceGetOptions)
	assert.Equal(deviceGetOptions.Namespace, deviceDefaultNamespace)
	assert.Equal(deviceGetOptions.PrintFlags, get.NewGetPrintFlags())
	assert.Equal(deviceGetOptions.PrintFlags.OutputFormat, &deviceGetOptions.Output)
}

func TestAddGetDeviceFlags(t *testing.T) {
	assert := assert.New(t)
	deviceGetOptions := NewDeviceGetOpts()

	cmd := &cobra.Command{}

	AddGetDeviceFlags(cmd, deviceGetOptions)

	namespaceFlag := cmd.Flags().Lookup(common.FlagNameNamespace)
	assert.NotNil(namespaceFlag)
	assert.Equal(deviceDefaultNamespace, namespaceFlag.DefValue)
	assert.Equal("namespace", namespaceFlag.Name)

	labelSelectorFlag := cmd.Flags().Lookup(common.FlagNameLabelSelector)
	assert.NotNil(labelSelectorFlag)
	assert.Equal("", labelSelectorFlag.DefValue)
	assert.Equal("selector", labelSelectorFlag.Name)

	outputFlag := cmd.Flags().Lookup(common.FlagNameOutput)
	assert.NotNil(outputFlag)
	assert.Equal("", outputFlag.DefValue)
	assert.Equal("output", outputFlag.Name)

	allNamespacesFlag := cmd.Flags().Lookup(common.FlagNameAllNamespaces)
	assert.NotNil(allNamespacesFlag)
	assert.Equal("false", allNamespacesFlag.DefValue)
	assert.Equal("all-namespaces", allNamespacesFlag.Name)
}

func TestGetDevicesErrorConfig(t *testing.T) {
	deviceGetOptions := NewDeviceGetOpts()

	patches := gomonkey.ApplyFunc(util.ParseEdgecoreConfig,
		func(configPath string) (*v1alpha2.EdgeCoreConfig, error) {
			return nil, errors.New("config parsing failed")
		})
	defer patches.Reset()

	err := deviceGetOptions.getDevices([]string{})

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "get edge config failed")
}

func TestGetDevicesNoResources(t *testing.T) {
	deviceGetOptions, _, patches := setupDeviceTest()
	defer patches.Reset()

	patches.ApplyMethod(reflect.TypeOf(&client.DeviceRequest{}), "GetDevices",
		func(_ *client.DeviceRequest, _ context.Context) (*v1beta1.DeviceList, error) {
			return &v1beta1.DeviceList{Items: []v1beta1.Device{}}, nil
		})

	patches.ApplyFunc(os.Stderr.Write,
		func(_ []byte) (int, error) {
			return 0, nil
		})

	err := deviceGetOptions.getDevices([]string{})

	assert.NoError(t, err)
}

func TestGetDevicesJSON(t *testing.T) {
	deviceGetOptions, _, patches := setupDeviceTest()
	deviceGetOptions.Output = "json"
	defer patches.Reset()

	patches.ApplyMethod(reflect.TypeOf(&client.DeviceRequest{}), "GetDevices",
		func(_ *client.DeviceRequest, _ context.Context) (*v1beta1.DeviceList, error) {
			return &v1beta1.DeviceList{
				Items: []v1beta1.Device{
					{
						Spec: v1beta1.DeviceSpec{
							NodeName: deviceTestNodeName,
						},
					},
				},
			}, nil
		})

	patches.ApplyMethod(reflect.TypeOf(&ctlcommon.ExtPrintFlags{}), "PrintToJSONYaml",
		func(_ *ctlcommon.ExtPrintFlags, _ []runtime.Object) error {
			return nil
		})

	err := deviceGetOptions.getDevices([]string{})

	assert.NoError(t, err)
}

func TestGetDeviceByName(t *testing.T) {
	deviceGetOptions, _, patches := setupDeviceTest()
	defer patches.Reset()

	patches.ApplyMethod(reflect.TypeOf(&client.DeviceRequest{}), "GetDevice",
		func(_ *client.DeviceRequest, _ context.Context) (*v1beta1.Device, error) {
			return &v1beta1.Device{
				Spec: v1beta1.DeviceSpec{
					NodeName: deviceTestNodeName,
				},
			}, nil
		})

	patches.ApplyMethod(reflect.TypeOf(&ctlcommon.ExtPrintFlags{}), "PrintToTable",
		func(_ *ctlcommon.ExtPrintFlags, _ interface{}, _ bool, _ io.Writer) error {
			return nil
		})

	err := deviceGetOptions.getDevices([]string{testDeviceName})

	assert.NoError(t, err)
}

func TestDeviceNotOnNode(t *testing.T) {
	deviceGetOptions, _, patches := setupDeviceTest()
	defer patches.Reset()

	patches.ApplyMethod(reflect.TypeOf(&client.DeviceRequest{}), "GetDevice",
		func(_ *client.DeviceRequest, _ context.Context) (*v1beta1.Device, error) {
			return &v1beta1.Device{
				Spec: v1beta1.DeviceSpec{
					NodeName: "other-node",
				},
			}, nil
		})

	patches.ApplyFunc(os.Stderr.Write,
		func(_ []byte) (int, error) {
			return 0, nil
		})

	err := deviceGetOptions.getDevices([]string{testDeviceName})

	assert.NoError(t, err)
}

func TestGetDeviceError(t *testing.T) {
	deviceGetOptions, _, patches := setupDeviceTest()
	defer patches.Reset()

	patches.ApplyMethod(reflect.TypeOf(&client.DeviceRequest{}), "GetDevice",
		func(_ *client.DeviceRequest, _ context.Context) (*v1beta1.Device, error) {
			return nil, errors.New("device not found")
		})

	patches.ApplyFunc(os.Stderr.Write,
		func(_ []byte) (int, error) {
			return 0, nil
		})

	err := deviceGetOptions.getDevices([]string{testDeviceName})

	assert.NoError(t, err)
}
