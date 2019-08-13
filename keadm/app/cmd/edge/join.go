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
	"io"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	types "github.com/kubeedge/kubeedge/keadm/app/cmd/common"
	"github.com/kubeedge/kubeedge/keadm/app/cmd/util"
)

var (
	edgeJoinLongDescription = `
"keadm join" command bootstraps KubeEdge's worker node (at the edge) component.
It checks if the pre-requisites are installed already,
If not installed, this command will help in download,
install and execute on the host.
It will also connect with cloud component to receive 
further instructions and forward telemetry data from 
devices to cloud
`
	edgeJoinExample = `
keadm join --cloudcoreip=<ip address> --edgenodeid=<unique string as edge identifier>

  - For this command --cloudcoreip flag is a Mandatory option
  - This command will download and install the default version of pre-requisites and KubeEdge

keadm join --cloudcoreip=10.20.30.40 --edgenodeid=testing123 --kubeedge-version=0.2.1 --k8sserverip=50.60.70.80:8080 --interfacename eth0

- In case, any flag is used in a format like "--docker-version" or "--docker-version=" (without a value)
  then default versions shown in help will be chosen. 
  The versions for "--docker-version", "--kubernetes-version" and "--kubeedge-version" flags should be in the
  format "18.06.3", "1.14.0" and "0.2.1" respectively
`
)

// NewEdgeJoin returns KubeEdge edge join command.
func NewEdgeJoin(out io.Writer, joinOptions *types.JoinOptions) *cobra.Command {
	if joinOptions == nil {
		joinOptions = newJoinOptions()
	}

	tools := make(map[string]types.ToolsInstaller, 0)
	flagVals := make(map[string]types.FlagData, 0)

	cmd := &cobra.Command{
		Use:     "join",
		Short:   "Bootstraps edge component. Checks and install (if required) the pre-requisites. Execute it on any edge node machine you wish to join",
		Long:    edgeJoinLongDescription,
		Example: edgeJoinExample,
		RunE: func(cmd *cobra.Command, args []string) error {

			//Visit all the flags and store their values and default values.
			checkFlags := func(f *pflag.Flag) {
				util.AddToolVals(f, flagVals)
			}
			cmd.Flags().VisitAll(checkFlags)

			Add2ToolsList(tools, flagVals, joinOptions)
			return Execute(tools)
		},
	}

	addJoinOtherFlags(cmd, joinOptions)
	return cmd
}

func addJoinOtherFlags(cmd *cobra.Command, joinOptions *types.JoinOptions) {

	cmd.Flags().StringVar(&joinOptions.KubeEdgeVersion, types.KubeEdgeVersion, joinOptions.KubeEdgeVersion,
		"Use this key to download and use the required KubeEdge version")
	cmd.Flags().Lookup(types.KubeEdgeVersion).NoOptDefVal = joinOptions.KubeEdgeVersion

	cmd.Flags().StringVar(&joinOptions.DockerVersion, types.DockerVersion, joinOptions.DockerVersion,
		"Use this key to download and use the required Docker version")
	cmd.Flags().Lookup(types.DockerVersion).NoOptDefVal = joinOptions.DockerVersion

	cmd.Flags().StringVar(&joinOptions.InterfaceName, types.InterfaceName, joinOptions.InterfaceName,
		"KubeEdge Node interface name string, the default value is eth0")

	cmd.Flags().StringVarP(&joinOptions.K8SAPIServerIPPort, types.K8SAPIServerIPPort, "k", joinOptions.K8SAPIServerIPPort,
		"IP:Port address of K8S API-Server")

	cmd.Flags().StringVarP(&joinOptions.CloudCoreIP, types.CloudCoreIP, "e", joinOptions.CloudCoreIP,
		"IP address of KubeEdge CloudCore")
	cmd.MarkFlagRequired(types.CloudCoreIP)
	cmd.Flags().StringVarP(&joinOptions.RuntimeType, types.RuntimeType, "r", joinOptions.RuntimeType,
		"Container runtime type")
	cmd.Flags().StringVarP(&joinOptions.EdgeNodeID, types.EdgeNodeID, "i", joinOptions.EdgeNodeID,
		"KubeEdge Node unique identification string, If flag not used then the command will generate a unique id on its own")
}

// newJoinOptions returns a struct ready for being used for creating cmd join flags.
func newJoinOptions() *types.JoinOptions {
	opts := &types.JoinOptions{}
	opts.InitOptions = types.InitOptions{DockerVersion: types.DefaultDockerVersion, KubeEdgeVersion: types.DefaultKubeEdgeVersion,
		KubernetesVersion: types.DefaultK8SVersion}
	opts.CertPath = types.DefaultCertPath
	return opts
}

//Add2ToolsList Reads the flagData (containing val and default val) and join options to fill the list of tools.
func Add2ToolsList(toolList map[string]types.ToolsInstaller, flagData map[string]types.FlagData, joinOptions *types.JoinOptions) {

	var kubeVer, dockerVer string

	flgData, ok := flagData[types.KubeEdgeVersion]
	if ok {
		kubeVer = util.CheckIfAvailable(flgData.Val.(string), flgData.DefVal.(string))
	} else {
		kubeVer = joinOptions.KubeEdgeVersion
	}
	toolList["KubeEdge"] = &util.KubeEdgeInstTool{Common: util.Common{ToolVersion: kubeVer}, K8SApiServerIP: joinOptions.K8SAPIServerIPPort,
		CloudCoreIP: joinOptions.CloudCoreIP, EdgeNodeID: joinOptions.EdgeNodeID, RuntimeType: joinOptions.RuntimeType, InterfaceName: joinOptions.InterfaceName}

	flgData, ok = flagData[types.DockerVersion]
	if ok {
		dockerVer = util.CheckIfAvailable(flgData.Val.(string), flgData.DefVal.(string))
	} else {
		dockerVer = joinOptions.DockerVersion
	}
	toolList["Docker"] = &util.DockerInstTool{Common: util.Common{ToolVersion: dockerVer}, DefaultToolVer: flgData.DefVal.(string)}
	toolList["MQTT"] = &util.MQTTInstTool{}
}

//Execute the instalation for each tool and start edgecore
func Execute(toolList map[string]types.ToolsInstaller) error {

	//Install all the required pre-requisite tools
	for name, tool := range toolList {
		if name != "KubeEdge" {
			err := tool.InstallTools()
			if err != nil {
				return err
			}
		}
	}

	//Install and Start KubeEdge Node
	return toolList["KubeEdge"].InstallTools()
}
