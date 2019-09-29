/*
Copyright 2019 The Kubeedge Authors.

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

package cmd

import (
	"github.com/spf13/cobra"
	//"github.com/spf13/viper"
	"k8s.io/klog"

	types "github.com/kubeedge/kubeedge/keadm/app/cmd/common"
	"github.com/kubeedge/kubeedge/keadm/app/cmd/util"

	"fmt"
	"io"
	"os"
	"os/exec"
)

var (
	startLongDescription = `
"keadm start" command will start KubeEdge's worker node (at the edge) component.
It will also connect the edge component with the cloud component to recieve further
instructions and forward telemetry data from devices to cloud
`

	startExample = `
keadm start <part>

	- For this command <part> is mandatory; Currently two options with cloud and edge are allowed

keadm start <part> --dir=<kubeedge directory>

	- For this command <part> is mandatory; Currently two options with cloud and edge are allowed
	- For this command --dir flag is a optional option the default value is /etc/kubeedge
`

	startShortDescription = "this command will start a speciied kubeEdge part"
)

// Start will execute the start command
func Start(out io.Writer, startOptions *types.StartOptions) *cobra.Command {
	klog.InitFlags(nil)
	var d string
	if startOptions == nil {
		startOptions = newStartOption()
	}

	cmd := &cobra.Command{
		Use:     "start",
		Short:   startShortDescription,
		Long:    startLongDescription,
		Example: startExample,
	}

	edge := &cobra.Command{
		Use:     "edge",
		Short:   "start edge",
		Example: "sudo keadm start edge\nsudo keadm start edge --dir /etc/kubeedge",
		RunE: func(cmd *cobra.Command, args []string) error {
			dr, err := cmd.Flags().GetString("dir")
			if err != nil {
				klog.Errorf("could not parse flag dir; err: %v\n", err)
			}

			for _, arg := range args {
				klog.Infof("arg %v\n", arg)
			}
			return startEdge(dr)
		},
	}

	edge.PersistentFlags().StringVarP(&d, "dir", "d", "/etc/kubeedge", "set working directory default=/etc/kubeedge")
	cloud := &cobra.Command{
		Use:     "cloud",
		Short:   "start edge",
		Example: "sudo keadm start cloud\nsudo keadm start cloud --dir /etc/kubeedge",
		RunE: func(cmd *cobra.Command, args []string) error {
			dir, err := cmd.Flags().GetString("dir")
			if err != nil {
				klog.Errorf("could not parse flag dir; err: %v\n", err)
			}
			return startCloud(dir)
		},
	}
	cloud.PersistentFlags().StringVarP(&d, "dir", "d", "/etc/kubeedge", "set working directory default=/etc/kubeedge")

	cmd.AddCommand(edge)
	cmd.AddCommand(cloud)

	return cmd
}

func newStartOption() *types.StartOptions {
	opts := &types.StartOptions{}
	opts.Dir = "/etc/kubeedge"
	return opts
}

func startEdge(dir string) error {
	// in a feature version we could check the configuration here
	klog.Infof("using working dir: %s\n", dir)
	binExec := fmt.Sprintf("%s > %s/kubeedge/edge/%s.log 2>&1 &", util.KubeEdgeBinaryName, util.KubeEdgePath, util.KubeEdgeBinaryName)
	cmd := exec.Command(binExec)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), fmt.Sprintf("GOARCHAIUS_CONFIG_PATH=%skubeedge/edge", util.KubeEdgePath))
	if err := cmd.Run(); err != nil {
		klog.Errorf("could not execute command; err: %v\n", err)
		return err
	}

	return nil
}

func startCloud(dir string) error {
	// in a feature version we could check the configuration here
	klog.Infof("using working dir: %s\n", dir)
	binExec := fmt.Sprintf("%s > %s/kubeedge/cloud/%s.log 2>&1 &", util.KubeCloudBinaryName, util.KubeEdgePath, util.KubeCloudBinaryName)
	cmd := exec.Command(binExec)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), fmt.Sprintf("GOARCHAIUS_CONFIG_PATH=%skubeedge/cloud", util.KubeEdgePath))
	if err := cmd.Run(); err != nil {
		klog.Errorf("could not execute command; err: %v\n", err)
		return err
	}

	return nil
}
