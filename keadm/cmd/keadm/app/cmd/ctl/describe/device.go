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

	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/cli-runtime/pkg/genericiooptions"
	cmdutil "k8s.io/kubectl/pkg/cmd/util"
	"k8s.io/kubectl/pkg/describe"

	"github.com/kubeedge/api/apis/devices/v1beta1"
	"github.com/kubeedge/kubeedge/keadm/cmd/keadm/app/cmd/common"
	"github.com/kubeedge/kubeedge/keadm/cmd/keadm/app/cmd/ctl/client"
	"github.com/kubeedge/kubeedge/keadm/cmd/keadm/app/cmd/util"
	"github.com/spf13/cobra"
)

var edgeDescribeDeviceShortDescription = `Describe devices in edge node`

type DescribeDeviceOptions struct {
	Namespace     string
	LabelSelector string
	AllNamespaces bool
	ShowEvents    bool
	ChunkSize     int64
	genericiooptions.IOStreams
}

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
	}
	AddDescribeDeviceFlags(cmd, describeDeviceOptions)
	return cmd
}

func NewDescribeDeviceOptions() *DescribeDeviceOptions {
	return &DescribeDeviceOptions{
		IOStreams: genericiooptions.IOStreams{In: os.Stdin, Out: os.Stdout, ErrOut: os.Stderr}}
}

func AddDescribeDeviceFlags(cmd *cobra.Command, options *DescribeDeviceOptions) {
	cmd.Flags().StringVarP(&options.Namespace, "namespace", "n", "default", "If present, the namespace scope for this CLI request")
	cmd.Flags().StringVar(&options.LabelSelector, "selector", "", "Selector (label query) to filter on")
	cmd.Flags().BoolVar(&options.AllNamespaces, "all-namespaces", false, "If present, list the requested object(s) across all namespaces")
	cmd.Flags().BoolVar(&options.ShowEvents, "show-events", false, "If present, list the requested object(s) across all namespaces")
	cmd.Flags().Int64Var(&options.ChunkSize, "chunk-size", 500, "If non-zero, split output into chunks of this many bytes")
}

func (o *DescribeDeviceOptions) describeDevice(args []string) error {
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
				fmt.Println(err.Error())
				continue
			}
			if device.Spec.NodeName == nodeName {
				deviceListFilter.Items = append(deviceListFilter.Items, *device)
			} else {
				fmt.Printf("can't query device: \"%s\" for node: \"%s\"\n", device.Name, device.Spec.NodeName)
			}
		}
	} else {
		deviceRequest := &client.DeviceRequest{
			Namespace:     o.Namespace,
			LabelSelector: o.LabelSelector,
			AllNamespaces: o.AllNamespaces,
		}

		deviceList, err := deviceRequest.GetFakeDevices(ctx)
		// Bug in list devices. It should be GetDevices instead of GetFakeDevices
		// deviceList, err := deviceRequest.GetDevices(ctx)
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
			fmt.Println("No resources found in all namespaces.")
		} else {
			fmt.Printf("No resources found in %s namespaces.\n", o.Namespace)
		}

		return nil
	}

	NamespaceToDeviceName := make(map[string]string)

	for _, device := range deviceListFilter.Items {
		NamespaceToDeviceName[device.Namespace] = device.Name
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
	for namespace, deviceName := range NamespaceToDeviceName {
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
			fmt.Fprint(o.Out, s)
		} else {
			fmt.Fprintf(o.Out, "\n\n%s", s)
		}
	}

	return nil
}
