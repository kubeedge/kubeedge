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
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/cli-runtime/pkg/genericiooptions"
	cmdutil "k8s.io/kubectl/pkg/cmd/util"
	"k8s.io/kubectl/pkg/cmd/util/editor"
	"sigs.k8s.io/yaml"

	"github.com/kubeedge/api/apis/devices/v1beta1"
	"github.com/kubeedge/kubeedge/keadm/cmd/keadm/app/cmd/common"
	"github.com/kubeedge/kubeedge/keadm/cmd/keadm/app/cmd/ctl/client"
	"github.com/kubeedge/kubeedge/keadm/cmd/keadm/app/cmd/util"
	"github.com/spf13/cobra"
)

var edgeEditDeviceShortDescription = `Edit a device in edge node`

type EditDeviceOptions struct {
	Namespace string

	genericiooptions.IOStreams
	editPrinterOptions *editPrinterOptions
}

type editPrinterOptions struct {
	printFlags *genericclioptions.PrintFlags
	ext        string
	addHeader  bool
}

func NewEdgeEditDevice() *cobra.Command {
	editDeviceOptions := NewEditDeviceOpts()
	cmd := &cobra.Command{
		Use:   "device",
		Short: edgeEditDeviceShortDescription,
		Long:  edgeEditDeviceShortDescription,
		RunE: func(cmd *cobra.Command, args []string) error {
			cmdutil.CheckErr(editDeviceOptions.editDevice(args))
			return nil
		},
	}
	AddEditDeviceFlags(cmd, editDeviceOptions)
	return cmd
}

func NewEditDeviceOpts() *EditDeviceOptions {
	return &EditDeviceOptions{
		IOStreams: genericiooptions.IOStreams{In: os.Stdin, Out: os.Stdout, ErrOut: os.Stderr},
		editPrinterOptions: &editPrinterOptions{
			printFlags: (&genericclioptions.PrintFlags{
				JSONYamlPrintFlags: genericclioptions.NewJSONYamlPrintFlags(),
			}).WithDefaultOutput("yaml"),
			ext:       ".yaml",
			addHeader: false,
		},
	}
}

func (o *EditDeviceOptions) editDevice(args []string) error {
	config, err := util.ParseEdgecoreConfig(common.EdgecoreConfigPath)
	if err != nil {
		return fmt.Errorf("get edge config failed with err:%v", err)
	}
	nodeName := config.Modules.Edged.HostnameOverride

	ctx := context.Background()

	if len(args) == 1 {
		deviceRequest := &client.DeviceRequest{
			Namespace:  o.Namespace,
			DeviceName: args[0],
		}

		device, err := deviceRequest.GetDevice(ctx)
		if err != nil {
			return err
		}

		if device.Spec.NodeName == nodeName {
			if err = o.edit(device); err != nil {
				return err
			}
			fmt.Println("Send update message to DeviceTwin")
		} else {
			fmt.Printf("Can't query device: \"%s\" for node: \"%s\"\n", device.Name, device.Spec.NodeName)
		}
	} else {
		return fmt.Errorf("too many args, edit one device at once")
	}

	return nil
}

func (e *editPrinterOptions) PrintObj(obj *v1beta1.Device, out io.Writer) error {
	// TODO: only yaml format is supported to print information,
	// and other formats such as json are to be implemented
	obj.GetObjectKind().SetGroupVersionKind(v1beta1.SchemeGroupVersion.WithKind("Device"))

	jsonData, _ := json.Marshal(*obj)
	data, err := yaml.JSONToYAML(jsonData)
	if err != nil {
		return err
	}

	_, err = out.Write(data)
	return err
}

func (o *EditDeviceOptions) edit(dl *v1beta1.Device) error {
	edit := editor.NewDefaultEditor([]string{
		"KUBE_EDITOR",
		"EDITOR",
	})
	buf := &bytes.Buffer{}
	var w io.Writer = buf

	if err := o.editPrinterOptions.PrintObj(dl, w); err != nil {
		return err
	}
	original := buf.Bytes()
	edited, file, err := edit.LaunchTempFile(fmt.Sprintf("%s-edit-", filepath.Base(os.Args[0])), "", buf)
	if err != nil {
		return preservedFile(err, file)
	}

	if bytes.Equal(cmdutil.StripComments(original), cmdutil.StripComments(edited)) {
		os.Remove(file)
		fmt.Println("Edit cancelled, no changes made.")
		return nil
	}

	jsonEdited := cmdutil.StripComments(edited)

	var editedDevice v1beta1.Device
	err = json.Unmarshal(jsonEdited, &editedDevice)
	if err != nil {
		return preservedFile(err, file)
	}

	deviceRequest := &client.DeviceRequest{
		Namespace:  dl.Namespace,
		DeviceName: dl.Name,
	}

	_, err = deviceRequest.UpdateDevice(context.Background(), &editedDevice)
	if err != nil {
		return preservedFile(err, file)
	}

	return nil
}

func preservedFile(err error, path string) error {
	if len(path) > 0 {
		if _, err := os.Stat(path); !os.IsNotExist(err) {
			fmt.Printf("A copy of your changes has been stored to %q\n", path)
		}
	}
	return err
}

func AddEditDeviceFlags(cmd *cobra.Command, o *EditDeviceOptions) {
	cmd.Flags().StringVarP(&o.Namespace, "namespace", "n", "default", "If present, the namespace scope for this CLI request")
}
