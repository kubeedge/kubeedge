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

	types "github.com/kubeedge/kubeedge/keadm/app/cmd/common"
	"github.com/kubeedge/kubeedge/keadm/app/cmd/util"
)

var (
	cloudInitLongDescription = `
"keadm init" command bootstraps KubeEdge's master node (on the cloud) component.
It checks if the pre-requisites are installed already,
If not installed, this command will help in download,
install and execute on the host.
`
	cloudInitExample = `
keadm init

- This command will download and install the default version of pre-requisites and KubeEdge

keadm init --kubeedge-version=0.2.1 --kubernetes-version=1.14.1 --docker-version=18.06.3 --kube-config=~/.kube/config

  - In case, any flag is used in a format like "--docker-version" or "--docker-version=" (without a value)
    then default versions shown in help will be choosen. 
    The versions for "--docker-version", "--kubernetes-version" and "--kubeedge-version" flags should be in the
    format "18.06.3", "1.14.0" and "0.2.1" respectively
  - kube-config is the path of kubeconfig which used to secure connectivity between edgecontroller and kube-apiserver
`
)

// NewCloudInit represents the keadm init command for cloud component
func NewCloudInit(out io.Writer, init *types.InitOptions) *cobra.Command {
	if init == nil {
		init = newInitOptions()
	}
	tools := make(map[string]types.ToolsInstaller, 0)
	flagVals := make(map[string]types.FlagData, 0)

	var cmd = &cobra.Command{
		Use:     "init",
		Short:   "Bootstraps cloud component. Checks and install (if required) the pre-requisites.",
		Long:    cloudInitLongDescription,
		Example: cloudInitExample,
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
func newInitOptions() *types.InitOptions {
	var opts *types.InitOptions
	opts = &types.InitOptions{}
	opts.DockerVersion = types.DefaultDockerVersion
	opts.KubeEdgeVersion = types.DefaultKubeEdgeVersion
	opts.KubernetesVersion = types.DefaultK8SVersion

	return opts
}

func addJoinOtherFlags(cmd *cobra.Command, initOpts *types.InitOptions) {

	cmd.Flags().StringVar(&initOpts.KubeEdgeVersion, types.KubeEdgeVersion, initOpts.KubeEdgeVersion,
		"Use this key to download and use the required KubeEdge version")
	cmd.Flags().Lookup(types.KubeEdgeVersion).NoOptDefVal = initOpts.KubeEdgeVersion

	cmd.Flags().StringVar(&initOpts.DockerVersion, types.DockerVersion, initOpts.DockerVersion,
		"Use this key to download and use the required Docker version")
	cmd.Flags().Lookup(types.DockerVersion).NoOptDefVal = initOpts.DockerVersion

	cmd.Flags().StringVar(&initOpts.KubernetesVersion, types.KubernetesVersion, initOpts.KubernetesVersion,
		"Use this key to download and use the required Kubernetes version")
	cmd.Flags().Lookup(types.KubernetesVersion).NoOptDefVal = initOpts.KubernetesVersion

	cmd.Flags().StringVar(&initOpts.KubeConfig, types.KubeConfig, initOpts.KubeConfig,
		"Use this key to set kube-config path, eg: $HOME/.kube/config")
}

//Add2ToolsList Reads the flagData (containing val and default val) and join options to fill the list of tools.
func Add2ToolsList(toolList map[string]types.ToolsInstaller, flagData map[string]types.FlagData, initOptions *types.InitOptions) {
	var kubeVer, dockerVer, k8sVer string

	flgData, ok := flagData[types.KubeEdgeVersion]
	if ok {
		kubeVer = util.CheckIfAvailable(flgData.Val.(string), flgData.DefVal.(string))
	} else {
		kubeVer = initOptions.KubeEdgeVersion
	}
	toolList["Cloud"] = &util.KubeCloudInstTool{Common: util.Common{ToolVersion: kubeVer, KubeConfig: initOptions.KubeConfig}}

	flgData, ok = flagData[types.DockerVersion]
	if ok {
		dockerVer = util.CheckIfAvailable(flgData.Val.(string), flgData.DefVal.(string))
	} else {
		dockerVer = initOptions.DockerVersion
	}
	toolList["Docker"] = &util.DockerInstTool{Common: util.Common{ToolVersion: dockerVer}, DefaultToolVer: flgData.DefVal.(string)}

	flgData, ok = flagData[types.KubernetesVersion]
	if ok {
		k8sVer = util.CheckIfAvailable(flgData.Val.(string), flgData.DefVal.(string))
	} else {
		k8sVer = initOptions.KubernetesVersion
	}
	toolList["Kubernetes"] = &util.K8SInstTool{Common: util.Common{ToolVersion: k8sVer}, IsEdgeNode: false, DefaultToolVer: flgData.DefVal.(string)}

}

//Execute the installation for each tool and start edgecontroller
func Execute(toolList map[string]types.ToolsInstaller) {

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
