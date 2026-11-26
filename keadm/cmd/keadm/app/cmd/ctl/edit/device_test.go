package edit

import (
	"bytes"
	"testing"

	"github.com/kubeedge/api/apis/devices/v1beta1"
	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/yaml"
)

func TestEditPrinterOptionsPrintObj(t *testing.T) {
	printer := &editPrinterOptions{}

	device := &v1beta1.Device{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "devices.kubeedge.io/v1beta1",
			Kind:       "Device",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: "device1",
		},
		Spec: v1beta1.DeviceSpec{
			NodeName: "edge-node",
		},
	}

	var buf bytes.Buffer

	err := printer.PrintObj(device, &buf)
	assert.NoError(t, err)

	output := buf.String()

	assert.Contains(t, output, "apiVersion:")
	assert.Contains(t, output, "kind: Device")
	assert.Contains(t, output, "name: device1")
	assert.Contains(t, output, "nodeName: edge-node")
}

func TestPrintObjOutputCanBeUnmarshaledAsYAML(t *testing.T) {
	printer := &editPrinterOptions{}

	device := &v1beta1.Device{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "devices.kubeedge.io/v1beta1",
			Kind:       "Device",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: "device1",
		},
		Spec: v1beta1.DeviceSpec{
			NodeName: "edge-node",
		},
	}

	var buf bytes.Buffer

	err := printer.PrintObj(device, &buf)
	assert.NoError(t, err)

	var editedDevice v1beta1.Device

	err = yaml.Unmarshal(buf.Bytes(), &editedDevice)
	assert.NoError(t, err)

	assert.Equal(t, "device1", editedDevice.Name)
	assert.Equal(t, "edge-node", editedDevice.Spec.NodeName)
}
