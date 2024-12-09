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

package describe

import (
	"context"
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/cli-runtime/pkg/genericiooptions"
	"k8s.io/klog/v2"
	cmdutil "k8s.io/kubectl/pkg/cmd/util"
	"k8s.io/kubectl/pkg/describe"

	"github.com/kubeedge/api/apis/devices/v1beta1"
	"github.com/kubeedge/kubeedge/keadm/cmd/keadm/app/cmd/common"
	"github.com/kubeedge/kubeedge/keadm/cmd/keadm/app/cmd/ctl/client"
	"github.com/kubeedge/kubeedge/keadm/cmd/keadm/app/cmd/util"
)

var edgeDescribeDeviceShortDescription = `Describe device in edge node`

type DeviceDescribeOptions struct {
	Namespace     string
	LabelSelector string
	AllNamespaces bool
	// ShowEvents is true if events should be included in the description. Default is false.
	ShowEvents bool
	// ChunkSize is the number of bytes to include in a chunk. Default is 500.
	ChunkSize int64
	genericiooptions.IOStreams
}

// NewEdgeDescribeDevice returns KubeEdge describe edge device command.
func NewEdgeDescribeDevice() *cobra.Command {
	describeDeviceOptions := NewDescribeDeviceOptions()
	cmd := &cobra.Command{
		Use:   "device",
		Short: edgeDescribeDeviceShortDescription,
		Long:  edgeDescribeDeviceShortDescription,
		RunE: func(cmd *cobra.Command, args []string) error {
			cmdutil.CheckErr(describeDeviceOptions.describeDevice(args))
			return nil
		},
		Aliases: []string{"devices"},
	}
	AddDescribeDeviceFlags(cmd, describeDeviceOptions)
	return cmd
}

func NewDescribeDeviceOptions() *DeviceDescribeOptions {
	return &DeviceDescribeOptions{
		IOStreams: genericiooptions.IOStreams{In: os.Stdin, Out: os.Stdout, ErrOut: os.Stderr},
	}
}

func AddDescribeDeviceFlags(cmd *cobra.Command, options *DeviceDescribeOptions) {
	cmd.Flags().StringVarP(&options.Namespace, common.FlagNameNamespace, "n", "default", "If present, the namespace scope for this CLI request")
	cmd.Flags().StringVar(&options.LabelSelector, common.FlagNameLabelSelector, "", "Selector (label query) to filter on")
	cmd.Flags().BoolVarP(&options.AllNamespaces, common.FlagNameAllNamespaces, "A", false, "If present, list the requested object(s) across all namespaces")
	cmd.Flags().BoolVar(&options.ShowEvents, common.FlagNameShowEvents, false, "If present, list the requested object(s) across all namespaces")
	cmd.Flags().Int64Var(&options.ChunkSize, common.FlagNameChunkSize, 500, "If non-zero, split output into chunks of this many bytes")
}

func (o *DeviceDescribeOptions) describeDevice(args []string) error {
	config, err := util.ParseEdgecoreConfig(common.EdgecoreConfigPath)
	if err != nil {
		return fmt.Errorf("get edge config failed with err:%v", err)
	}
	nodeName := config.Modules.Edged.HostnameOverride

	ctx := context.Background()

	var deviceListFilter *v1beta1.DeviceList

	if len(args) > 0 {
		deviceListFilter = &v1beta1.DeviceList{
			Items: make([]v1beta1.Device, 0, len(args)),
		}

		var deviceRequest *client.DeviceRequest
		for _, deviceName := range args {
			deviceRequest = &client.DeviceRequest{
				Namespace:  o.Namespace,
				DeviceName: deviceName,
			}
			device, err := deviceRequest.GetDevice(ctx)
			if err != nil {
				klog.Error(err.Error())
				continue
			}
			if device.Spec.NodeName == nodeName {
				deviceListFilter.Items = append(deviceListFilter.Items, *device)
			} else {
				klog.Errorf("can't query device: \"%s\" for node: \"%s\"", device.Name, device.Spec.NodeName)
			}
		}
	} else {
		deviceRequest := &client.DeviceRequest{
			Namespace:     o.Namespace,
			LabelSelector: o.LabelSelector,
			AllNamespaces: o.AllNamespaces,
		}

		deviceList, err := deviceRequest.GetDevices(ctx)
		if err != nil {
			return err
		}

		deviceListFilter = &v1beta1.DeviceList{
			Items: make([]v1beta1.Device, 0, len(deviceList.Items)),
		}

		for _, device := range deviceList.Items {
			if device.Spec.NodeName == nodeName {
				deviceListFilter.Items = append(deviceListFilter.Items, device)
			}
		}
	}

	if len(deviceListFilter.Items) == 0 {
		if len(args) > 0 {
			return nil
		}

		if o.AllNamespaces {
			klog.Info("No resources found in all namespaces.")
		} else {
			klog.Infof("No resources found in %s namespaces.", o.Namespace)
		}

		return nil
	}

	NamespaceToDeviceName := make(map[string][]string)

	for _, device := range deviceListFilter.Items {
		if _, ok := NamespaceToDeviceName[device.Namespace]; !ok {
			NamespaceToDeviceName[device.Namespace] = make([]string, 0)
		}
		NamespaceToDeviceName[device.Namespace] = append(NamespaceToDeviceName[device.Namespace], device.Name)
	}

	m := &meta.RESTMapping{
		Resource:         v1beta1.SchemeGroupVersion.WithResource("devices"),
		GroupVersionKind: v1beta1.SchemeGroupVersion.WithKind("Device"),
		Scope:            meta.RESTScopeNamespace,
	}
	c, err := client.GetKubeConfig()
	if err != nil {
		return err
	}
	d, ok := describe.GenericDescriberFor(m, c)
	if !ok {
		return fmt.Errorf("unable to find describer for %v", m)
	}

	first := true
	for namespace, deviceNameList := range NamespaceToDeviceName {
		for _, deviceName := range deviceNameList {
			settings := describe.DescriberSettings{
				ShowEvents: o.ShowEvents,
				ChunkSize:  o.ChunkSize,
			}
			s, err := d.Describe(namespace, deviceName, settings)
			if err != nil {
				return err
			}

			if first {
				first = false
				klog.Info(s)
			} else {
				klog.Infof("\n\n%s", s)
			}
		}
	}

	return nil
}
