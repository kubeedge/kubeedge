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

package get

import (
	"context"
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/cli-runtime/pkg/printers"
	"k8s.io/klog/v2"
	"k8s.io/kubectl/pkg/cmd/get"
	cmdutil "k8s.io/kubectl/pkg/cmd/util"
	"k8s.io/kubectl/pkg/scheme"

	"github.com/kubeedge/api/apis/devices/v1beta1"
	"github.com/kubeedge/kubeedge/keadm/cmd/keadm/app/cmd/common"
	"github.com/kubeedge/kubeedge/keadm/cmd/keadm/app/cmd/ctl/client"
	"github.com/kubeedge/kubeedge/keadm/cmd/keadm/app/cmd/util"
)

var edgeDeviceGetShortDescription = `Get devices in edge node`

type DeviceGetOptions struct {
	Namespace     string
	LabelSelector string
	AllNamespaces bool
	Output        string
	// PrintFlags holds the flags for printing resources
	PrintFlags *get.PrintFlags
}

// NewEdgeDeviceGet returns KubeEdge get edge device command.
func NewEdgeDeviceGet() *cobra.Command {
	deviceGetOptions := NewDeviceGetOpts()
	cmd := &cobra.Command{
		Use:   "device",
		Short: edgeDeviceGetShortDescription,
		Long:  edgeDeviceGetShortDescription,
		RunE: func(cmd *cobra.Command, args []string) error {
			cmdutil.CheckErr(deviceGetOptions.getDevices(args))
			return nil
		},
		Aliases: []string{"devices"},
	}

	AddGetDeviceFlags(cmd, deviceGetOptions)
	return cmd
}

func (o *DeviceGetOptions) getDevices(args []string) error {
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
			klog.Infof("No resources found in %s namespace.", o.Namespace)
		}
		return nil
	}

	if o.AllNamespaces {
		if err := o.PrintFlags.EnsureWithNamespace(); err != nil {
			return err
		}
	}

	o.PrintFlags.SetKind(v1beta1.SchemeGroupVersion.WithKind("DeviceList").GroupKind())

	printer, err := o.PrintFlags.ToPrinter()
	if err != nil {
		return err
	}

	printer, err = printers.NewTypeSetter(scheme.Scheme).WrapToPrinter(printer, nil)
	if err != nil {
		return nil
	}

	var deviceObjectList runtime.Object = deviceListFilter
	deviceObjectList.GetObjectKind().SetGroupVersionKind(v1beta1.SchemeGroupVersion.WithKind("DeviceList"))

	return printer.PrintObj(deviceObjectList, os.Stdout)
}

func NewDeviceGetOpts() *DeviceGetOptions {
	deviceGetOptions := &DeviceGetOptions{}
	deviceGetOptions.Namespace = "default"
	deviceGetOptions.PrintFlags = get.NewGetPrintFlags()
	deviceGetOptions.PrintFlags.OutputFormat = &deviceGetOptions.Output
	return deviceGetOptions
}

func AddGetDeviceFlags(cmd *cobra.Command, deviceGetOptions *DeviceGetOptions) {
	cmd.Flags().StringVarP(&deviceGetOptions.Namespace, common.FlagNameNamespace, "n", deviceGetOptions.Namespace,
		"Specify a namespace")
	cmd.Flags().StringVarP(&deviceGetOptions.LabelSelector, common.FlagNameLabelSelector, "l", deviceGetOptions.LabelSelector,
		"Selector (label query) to filter on, supports '=', '==', and '!='.(e.g. -l key1=value1,key2=value2)")
	cmd.Flags().BoolVarP(&deviceGetOptions.AllNamespaces, common.FlagNameAllNamespaces, "A", deviceGetOptions.AllNamespaces,
		"If present, list the requested object(s) across all namespaces. Namespace in current context is ignored even if specified with --namespace")
}
