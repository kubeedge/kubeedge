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
	"strings"

	"github.com/kubeedge/kubeedge/kectl/app/cmd/options"
	"github.com/kubeedge/kubeedge/kectl/app/cmd/util"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

var (
	cloudInitLongDescription = `
kectl init command bootstraps KubeEdge's cloud component.
It checks if the pre-requisites are installed already,
If not installed, this command will help in download,
install and execute on the host.
`
	cloudInitExample = `

kectl cloud init  --docker-version= --kubernetes-version=<version> --kubeedge-version=<version>
`
)

type FlagData struct {
	Val    interface{}
	DefVal interface{}
}

// NewCloudInit represents the kubeedge cloud init command
func NewCloudInit(out io.Writer, init *options.InitOptions) *cobra.Command {
	if init == nil {
		init = newInitOptions()
	}
	tools := make(map[string]util.ToolsInstaller, 0)
	flagVals := make(map[string]FlagData, 0)
	var cmd = &cobra.Command{
		Use:     "init",
		Short:   "Bootstraps cloud component. Checks and install (if required) the pre-requisites.",
		Long:    cloudInitLongDescription,
		Example: cloudInitExample,
		Run: func(cmd *cobra.Command, args []string) {
			// TODO: Work your own magic here
			checkFlags := func(f *pflag.Flag) {
				AddToolVals(f, flagVals)
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

	cmd.Flags().StringVar(&initOpts.KubeEdgeVersion, options.KubeedgeVersion, initOpts.KubeEdgeVersion,
		"Use this key to download and use the required KubeEdge version")
	cmd.Flags().StringVar(&initOpts.DockerVersion, options.DockerVersion, initOpts.DockerVersion,
		"Use this key to download and use the required Docker version")
	cmd.Flags().StringVar(&initOpts.KubernetesVersion, options.Kubernetesversion, initOpts.KubernetesVersion,
		"Use this key to download and use the required Kubernetes version")
}

func Add2ToolsList(toolList map[string]util.ToolsInstaller, flagData map[string]FlagData, initOptions *options.InitOptions) {
	var kubeVer, dockerVer, k8sVer string

	flgData, ok := flagData[options.KubeedgeVersion]
	if ok {
		fmt.Println(options.KubeedgeVersion, "VAL:", flgData.Val.(string), "DEFVAL:", flgData.DefVal.(string))
		kubeVer = CheckIfAvailable(flgData.Val.(string), flgData.DefVal.(string))
	} else {
		kubeVer = initOptions.KubeEdgeVersion
	}
	toolList["Cloud"] = &util.KubeCloudInstTool{Common: util.Common{ToolVersion: kubeVer}}

	flgData, ok = flagData[options.DockerVersion]
	if ok {
		fmt.Println(options.DockerVersion, "VAL:", flgData.Val.(string), "DEFVAL:", flgData.DefVal.(string))
		dockerVer = CheckIfAvailable(flgData.Val.(string), flgData.DefVal.(string))
	} else {
		dockerVer = initOptions.DockerVersion
	}
	toolList["Docker"] = &util.DockerInstTool{Common: util.Common{ToolVersion: dockerVer}, DefaultToolVer: flgData.DefVal.(string)}

	flgData, ok = flagData[options.Kubernetesversion]
	if ok {
		fmt.Println(options.Kubernetesversion, "VAL:", flgData.Val.(string), "DEFVAL:", flgData.DefVal.(string))
		k8sVer = CheckIfAvailable(flgData.Val.(string), flgData.DefVal.(string))
	} else {
		k8sVer = initOptions.KubernetesVersion
	}
	toolList["Kubernetes"] = &util.K8SInstTool{Common: util.Common{ToolVersion: k8sVer}, IsEdgeNode: false, DefaultToolVer: flgData.DefVal.(string)}

}

func AddToolVals(f *pflag.Flag, flagData map[string]FlagData) {
	flagData[f.Name] = FlagData{Val: f.Value.String(), DefVal: f.DefValue}
}

//Install all the required tools
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

func CheckIfAvailable(val, deval string) string {
	if val == "" {
		return deval
	}
	return val
}
