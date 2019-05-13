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
	"fmt"
	"io"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	"github.com/kubeedge/kubeedge/keadm/app/cmd/options"
	"github.com/kubeedge/kubeedge/keadm/app/cmd/util"
)

var (
	cloudInitLongDescription = `
kubeedge init command bootstraps KubeEdge's cloud component.
It checks if the pre-requisites are installed already,
If not installed, this command will help in download,
install and execute on the host.
`
	cloudInitExample = `
kubeedge init
`
)

// NewCloudInit represents the kubeedge init command for cloud component
func NewCloudInit(out io.Writer, init *options.InitOptions) *cobra.Command {
	if init == nil {
		init = newInitOptions()
	}
	tools := make(map[string]util.ToolsInstaller, 0)
	flagVals := make(map[string]util.FlagData, 0)

	var cmd = &cobra.Command{
		Use:     "init",
		Short:   "Bootstraps cloud component. Checks and install (if required) the pre-requisites.",
		Long:    cloudInitLongDescription,
		Example: cloudInitExample,
		PreRunE: func(cmd *cobra.Command, args []string) error {

			whoRunning, err := util.IsKubeEdgeController()
			if err != nil {
				return err
			}
			if util.KubeEdgeEdgeRunning == whoRunning {
				return fmt.Errorf("This is KubeEdge Edge node, KubeEdge Cloud node should't be initialised in it")
			}
			return nil
		},
		Run: func(cmd *cobra.Command, args []string) {
			checkFlags := func(f *pflag.Flag) {
				util.AddToolVals(f, flagVals)
			}
			cmd.Flags().VisitAll(checkFlags)
			Add2ToolsList(tools, flagVals, init)
			Execute(tools)
		},
	}

	addJoinOtherFlags(cmd, init)
	return cmd
}

//newInitOptions will initialise new instance of options everytime
func newInitOptions() *options.InitOptions {
	var opts *options.InitOptions
	opts = &options.InitOptions{}
	opts.DockerVersion = options.DefaultDockerVersion
	opts.KubeEdgeVersion = options.DefaultKubeEdgeVersion
	opts.KubernetesVersion = options.DefaultK8SVersion

	return opts
}

func addJoinOtherFlags(cmd *cobra.Command, initOpts *options.InitOptions) {

	cmd.Flags().StringVar(&initOpts.KubeEdgeVersion, options.KubeEdgeVersion, initOpts.KubeEdgeVersion,
		"Use this key to download and use the required KubeEdge version")
	cmd.Flags().Lookup(options.KubeEdgeVersion).NoOptDefVal = initOpts.KubeEdgeVersion

	cmd.Flags().StringVar(&initOpts.DockerVersion, options.DockerVersion, initOpts.DockerVersion,
		"Use this key to download and use the required Docker version")
	cmd.Flags().Lookup(options.DockerVersion).NoOptDefVal = initOpts.DockerVersion

	cmd.Flags().StringVar(&initOpts.KubernetesVersion, options.KubernetesVersion, initOpts.KubernetesVersion,
		"Use this key to download and use the required Kubernetes version")
	cmd.Flags().Lookup(options.KubernetesVersion).NoOptDefVal = initOpts.KubernetesVersion
}

//Add2ToolsList Reads the flagData (containing val and default val) and join options to fill the list of tools.
func Add2ToolsList(toolList map[string]util.ToolsInstaller, flagData map[string]util.FlagData, initOptions *options.InitOptions) {
	var kubeVer, dockerVer, k8sVer string

	flgData, ok := flagData[options.KubeEdgeVersion]
	if ok {
		kubeVer = util.CheckIfAvailable(flgData.Val.(string), flgData.DefVal.(string))
	} else {
		kubeVer = initOptions.KubeEdgeVersion
	}
	toolList["Cloud"] = &util.KubeCloudInstTool{Common: util.Common{ToolVersion: kubeVer}}

	flgData, ok = flagData[options.DockerVersion]
	if ok {
		dockerVer = util.CheckIfAvailable(flgData.Val.(string), flgData.DefVal.(string))
	} else {
		dockerVer = initOptions.DockerVersion
	}
	toolList["Docker"] = &util.DockerInstTool{Common: util.Common{ToolVersion: dockerVer}, DefaultToolVer: flgData.DefVal.(string)}

	flgData, ok = flagData[options.KubernetesVersion]
	if ok {
		k8sVer = util.CheckIfAvailable(flgData.Val.(string), flgData.DefVal.(string))
	} else {
		k8sVer = initOptions.KubernetesVersion
	}
	toolList["Kubernetes"] = &util.K8SInstTool{Common: util.Common{ToolVersion: k8sVer}, IsEdgeNode: false, DefaultToolVer: flgData.DefVal.(string)}

}

//Execute the instalation for each tool and start edgecontroller
func Execute(toolList map[string]util.ToolsInstaller) {

	for name, tool := range toolList {
		if name != "Cloud" {
			err := tool.InstallTools()
			if err != nil {
				fmt.Println(err.Error())
				continue
			}
		}
	}
	err := toolList["Cloud"].InstallTools()
	if err != nil {
		fmt.Println(err.Error())
	}

}
