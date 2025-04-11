package get

import (
	"context"
	"fmt"
	"io"
	"os"
	"testing"

	"github.com/agiledragon/gomonkey/v2"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/kubectl/pkg/cmd/get"

	"github.com/kubeedge/api/apis/componentconfig/edgecore/v1alpha2"
	"github.com/kubeedge/api/apis/devices/v1beta1"
	"github.com/kubeedge/kubeedge/keadm/cmd/keadm/app/cmd/common"
	"github.com/kubeedge/kubeedge/keadm/cmd/keadm/app/cmd/ctl/client"
	"github.com/kubeedge/kubeedge/keadm/cmd/keadm/app/cmd/util"
)

const (
	testDeviceNodeName = "test-node"
)

func resetDeviceTestMode() {
	testDeviceMode = false
	testDeviceEdgeNodeName = ""
	testGetDeviceFunc = nil
	testGetDevicesFunc = nil
	testDeviceConfigError = nil
}

// createMockEdgeCoreConfig returns a properly structured EdgeCoreConfig for testing
func createMockDeviceEdgeCoreConfig() *v1alpha2.EdgeCoreConfig {
	config := v1alpha2.NewDefaultEdgeCoreConfig()
	config.Modules.Edged.HostnameOverride = testDeviceNodeName
	return config
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
	assert.Equal(defaultNamespace, deviceGetOptions.Namespace)
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
	assert.Equal(defaultNamespace, namespaceFlag.DefValue)
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

func captureDeviceOutput(f func()) string {
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	f()

	w.Close()
	os.Stdout = oldStdout

	outBytes, _ := io.ReadAll(r)
	return string(outBytes)
}

func createTestDevice(name, namespace, nodeName string) *v1beta1.Device {
	device := &v1beta1.Device{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Device",
			APIVersion: "devices.kubeedge.io/v1beta1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: v1beta1.DeviceSpec{
			NodeName: nodeName,
		},
	}

	gvk := schema.GroupVersionKind{
		Group:   "devices.kubeedge.io",
		Version: "v1beta1",
		Kind:    "Device",
	}
	device.GetObjectKind().SetGroupVersionKind(gvk)

	return device
}

func TestGetDevicesWithSingleDevice(t *testing.T) {
	defer resetDeviceTestMode()

	assert := assert.New(t)

	testDeviceMode = true
	testDeviceEdgeNodeName = testDeviceNodeName

	testGetDeviceFunc = func(ctx context.Context, namespace, name string) (*v1beta1.Device, error) {
		return createTestDevice(name, namespace, testDeviceNodeName), nil
	}

	opts := NewDeviceGetOpts()

	output := captureDeviceOutput(func() {
		err := opts.getDevices([]string{"test-device"})
		assert.NoError(err)
	})

	assert.Contains(output, "NAME")
	assert.Contains(output, "test-device")
}

func TestGetDevicesWithDifferentNode(t *testing.T) {
	defer resetDeviceTestMode()

	assert := assert.New(t)

	testDeviceMode = true
	testDeviceEdgeNodeName = testDeviceNodeName

	testGetDeviceFunc = func(ctx context.Context, namespace, name string) (*v1beta1.Device, error) {
		return createTestDevice(name, namespace, "different-node"), nil
	}

	opts := NewDeviceGetOpts()

	output := captureDeviceOutput(func() {
		err := opts.getDevices([]string{"test-device"})
		assert.NoError(err)
	})

	assert.NotContains(output, "test-device")
}

func TestGetDevicesWithDeviceError(t *testing.T) {
	defer resetDeviceTestMode()

	assert := assert.New(t)

	testDeviceMode = true
	testDeviceEdgeNodeName = testDeviceNodeName

	testGetDeviceFunc = func(ctx context.Context, namespace, name string) (*v1beta1.Device, error) {
		return nil, fmt.Errorf("device not found")
	}

	opts := NewDeviceGetOpts()

	err := opts.getDevices([]string{"test-device"})
	assert.NoError(err) // Function continues despite individual device errors
}

func TestGetAllDevices(t *testing.T) {
	defer resetDeviceTestMode()

	assert := assert.New(t)

	testDeviceMode = true
	testDeviceEdgeNodeName = testDeviceNodeName

	testGetDevicesFunc = func(ctx context.Context, namespace, labelSelector string, allNamespaces bool) (*v1beta1.DeviceList, error) {
		return &v1beta1.DeviceList{
			Items: []v1beta1.Device{
				*createTestDevice("device1", namespace, testDeviceNodeName),
				*createTestDevice("device2", namespace, "different-node"),
			},
		}, nil
	}

	opts := NewDeviceGetOpts()

	output := captureDeviceOutput(func() {
		err := opts.getDevices([]string{})
		assert.NoError(err)
	})

	assert.Contains(output, "NAME")
	assert.Contains(output, "device1")
	assert.NotContains(output, "device2")
}

func TestGetDevicesWithConfigError(t *testing.T) {
	defer resetDeviceTestMode()

	assert := assert.New(t)

	testDeviceMode = true
	testDeviceConfigError = fmt.Errorf("config parse error")

	opts := NewDeviceGetOpts()

	err := opts.getDevices([]string{})
	assert.Error(err)
	assert.Contains(err.Error(), "config parse error")
}

func TestGetDevicesWithGetDevicesError(t *testing.T) {
	defer resetDeviceTestMode()

	assert := assert.New(t)

	testDeviceMode = true
	testDeviceEdgeNodeName = testDeviceNodeName

	testGetDevicesFunc = func(ctx context.Context, namespace, labelSelector string, allNamespaces bool) (*v1beta1.DeviceList, error) {
		return nil, fmt.Errorf("failed to list devices")
	}

	opts := NewDeviceGetOpts()

	err := opts.getDevices([]string{})
	assert.Error(err)
	assert.Contains(err.Error(), "failed to list devices")
}

func TestGetDevicesWithNoDevices(t *testing.T) {
	defer resetDeviceTestMode()

	t.Run("default namespace", func(t *testing.T) {
		assert := assert.New(t)

		testDeviceMode = true
		testDeviceEdgeNodeName = testDeviceNodeName

		testGetDevicesFunc = func(ctx context.Context, namespace, labelSelector string, allNamespaces bool) (*v1beta1.DeviceList, error) {
			return &v1beta1.DeviceList{
				Items: []v1beta1.Device{},
			}, nil
		}

		opts := NewDeviceGetOpts()
		opts.AllNamespaces = false
		opts.Namespace = defaultNamespace

		err := opts.getDevices([]string{})
		assert.NoError(err)
	})

	t.Run("all namespaces", func(t *testing.T) {
		assert := assert.New(t)

		testDeviceMode = true
		testDeviceEdgeNodeName = testDeviceNodeName

		testGetDevicesFunc = func(ctx context.Context, namespace, labelSelector string, allNamespaces bool) (*v1beta1.DeviceList, error) {
			return &v1beta1.DeviceList{
				Items: []v1beta1.Device{},
			}, nil
		}

		opts := NewDeviceGetOpts()
		opts.AllNamespaces = true

		err := opts.getDevices([]string{})
		assert.NoError(err)
	})
}

func TestGetDevicesWithJSONOutput(t *testing.T) {
	defer resetDeviceTestMode()

	assert := assert.New(t)

	testDeviceMode = true
	testDeviceEdgeNodeName = testDeviceNodeName

	testGetDevicesFunc = func(ctx context.Context, namespace, labelSelector string, allNamespaces bool) (*v1beta1.DeviceList, error) {
		return &v1beta1.DeviceList{
			Items: []v1beta1.Device{
				*createTestDevice("device1", namespace, testDeviceNodeName),
			},
		}, nil
	}

	opts := NewDeviceGetOpts()
	opts.Output = "json"
	opts.PrintFlags = get.NewGetPrintFlags()
	opts.PrintFlags.OutputFormat = &opts.Output

	output := captureDeviceOutput(func() {
		err := opts.getDevices([]string{})
		assert.NoError(err)
	})

	assert.Contains(output, "{")
	assert.Contains(output, "\"name\": \"device1\"")
}

func TestDeviceRealModeConfigError(t *testing.T) {
	defer resetDeviceTestMode()
	testDeviceMode = false

	patches := gomonkey.ApplyFunc(util.ParseEdgecoreConfig,
		func(configPath string) (*v1alpha2.EdgeCoreConfig, error) {
			return nil, fmt.Errorf("mocked config error")
		})
	defer patches.Reset()

	opts := NewDeviceGetOpts()
	err := opts.getDevices([]string{})

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "get edge config failed with err:mocked config error")
}

func TestDeviceDirectCoverageRealMode(t *testing.T) {
	origTestMode := testDeviceMode
	origEdgeNodeName := testDeviceEdgeNodeName
	origGetDeviceFunc := testGetDeviceFunc
	origGetDevicesFunc := testGetDevicesFunc
	origConfigError := testDeviceConfigError

	defer func() {
		testDeviceMode = origTestMode
		testDeviceEdgeNodeName = origEdgeNodeName
		testGetDeviceFunc = origGetDeviceFunc
		testGetDevicesFunc = origGetDevicesFunc
		testDeviceConfigError = origConfigError
	}()

	testDeviceMode = false

	p1 := gomonkey.ApplyFunc(util.ParseEdgecoreConfig,
		func(configPath string) (*v1alpha2.EdgeCoreConfig, error) {
			config := createMockDeviceEdgeCoreConfig()
			return config, nil
		})
	defer p1.Reset()

	p2 := gomonkey.ApplyFunc((*client.DeviceRequest).GetDevice,
		func(_ *client.DeviceRequest, _ context.Context) (*v1beta1.Device, error) {
			return createTestDevice("test-device", "default", testDeviceNodeName), nil
		})
	defer p2.Reset()

	p3 := gomonkey.ApplyFunc((*client.DeviceRequest).GetDevices,
		func(_ *client.DeviceRequest, _ context.Context) (*v1beta1.DeviceList, error) {
			return &v1beta1.DeviceList{
				Items: []v1beta1.Device{
					*createTestDevice("device1", "default", testDeviceNodeName),
				},
			}, nil
		})
	defer p3.Reset()

	opts := NewDeviceGetOpts()

	_ = captureDeviceOutput(func() {
		_ = opts.getDevices([]string{"test-device"})
	})

	_ = captureDeviceOutput(func() {
		_ = opts.getDevices([]string{})
	})
}

func TestDeviceRealModeMockRequestMethods(t *testing.T) {
	defer resetDeviceTestMode()
	testDeviceMode = false

	config := createMockDeviceEdgeCoreConfig()

	patches := gomonkey.ApplyFunc(util.ParseEdgecoreConfig,
		func(configPath string) (*v1alpha2.EdgeCoreConfig, error) {
			return config, nil
		})
	defer patches.Reset()

	t.Run("single device", func(t *testing.T) {
		p1 := gomonkey.ApplyFunc((*client.DeviceRequest).GetDevice,
			func(_ *client.DeviceRequest, _ context.Context) (*v1beta1.Device, error) {
				return createTestDevice("test-device", "default", testDeviceNodeName), nil
			})
		defer p1.Reset()

		opts := NewDeviceGetOpts()
		_ = opts.getDevices([]string{"test-device"})
	})

	t.Run("multiple devices", func(t *testing.T) {
		p2 := gomonkey.ApplyFunc((*client.DeviceRequest).GetDevices,
			func(_ *client.DeviceRequest, _ context.Context) (*v1beta1.DeviceList, error) {
				return &v1beta1.DeviceList{
					Items: []v1beta1.Device{
						*createTestDevice("device1", "default", testDeviceNodeName),
					},
				}, nil
			})
		defer p2.Reset()

		opts := NewDeviceGetOpts()
		_ = opts.getDevices([]string{})
	})
}
