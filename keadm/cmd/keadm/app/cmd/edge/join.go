/*
Copyright 2019 The KubeEdge Authors.

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

	types "github.com/kubeedge/kubeedge/keadm/cmd/keadm/app/cmd/common"
	"github.com/kubeedge/kubeedge/keadm/cmd/keadm/app/cmd/util"
)

var (
	edgeJoinLongDescription = `
"keadm join" command bootstraps KubeEdge's worker node (at the edge) component.
It will also connect with cloud component to receive 
further instructions and forward telemetry data from 
devices to cloud
`
	edgeJoinExample = `
keadm join --cloudcore-ipport=<ip:port address> --edgenode-name=<unique string as edge identifier>

  - For this command --cloudcore-ipport flag is a required option
  - This command will download and install the default version of pre-requisites and KubeEdge

keadm join --cloudcore-ipport=10.20.30.40:10000 --edgenode-name=testing123 --kubeedge-version=1.2.1
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

			err := Add2ToolsList(tools, flagVals, joinOptions)
			if err != nil {
				return err
			}
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

	cmd.Flags().StringVar(&joinOptions.InterfaceName, types.InterfaceName, joinOptions.InterfaceName,
		"KubeEdge Node interface name string, the default value is eth0")

	cmd.Flags().StringVar(&joinOptions.CertPath, types.CertPath, joinOptions.CertPath,
		"The certPath used by edgecore, the default value is /etc/kubeedge/certs")

	cmd.Flags().StringVarP(&joinOptions.CloudCoreIPPort, types.CloudCoreIPPort, "e", joinOptions.CloudCoreIPPort,
		"IP:Port address of KubeEdge CloudCore")
	cmd.MarkFlagRequired(types.CloudCoreIPPort)

	cmd.Flags().StringVarP(&joinOptions.RuntimeType, types.RuntimeType, "r", joinOptions.RuntimeType,
		"Container runtime type")

	cmd.Flags().StringVarP(&joinOptions.EdgeNodeName, types.EdgeNodeName, "i", joinOptions.EdgeNodeName,
		"KubeEdge Node unique identification string, If flag not used then the command will generate a unique id on its own")

	cmd.Flags().StringVarP(&joinOptions.RemoteRuntimeEndpoint, types.RemoteRuntimeEndpoint, "p", joinOptions.RemoteRuntimeEndpoint,
		"KubeEdge Edge Node RemoteRuntimeEndpoint string, If flag not set, it will use unix:///var/run/dockershim.sock")
}

// newJoinOptions returns a struct ready for being used for creating cmd join flags.
func newJoinOptions() *types.JoinOptions {
	opts := &types.JoinOptions{}
	opts.CertPath = types.DefaultCertPath

	return opts
}

//Add2ToolsList Reads the flagData (containing val and default val) and join options to fill the list of tools.
func Add2ToolsList(toolList map[string]types.ToolsInstaller, flagData map[string]types.FlagData, joinOptions *types.JoinOptions) error {
	var kubeVer string

	flgData, ok := flagData[types.KubeEdgeVersion]
	if ok {
		kubeVer = util.CheckIfAvailable(flgData.Val.(string), flgData.DefVal.(string))
	}
	if kubeVer == "" {
		var latestVersion string
		for i := 0; i < util.RetryTimes; i++ {
			latestVersion, err := util.GetLatestVersion()
			if err != nil {
				return err
			}
			if len(latestVersion) != 0 {
				kubeVer = latestVersion[1:]
				break
			}
		}
		if len(latestVersion) == 0 {
			fmt.Println("Failed to get the latest KubeEdge release version, will use default version")
			kubeVer = types.DefaultKubeEdgeVersion
		}
	}
	toolList["KubeEdge"] = &util.KubeEdgeInstTool{
		Common: util.Common{
			ToolVersion: kubeVer,
		},
		CloudCoreIP:           joinOptions.CloudCoreIPPort,
		EdgeNodeName:          joinOptions.EdgeNodeName,
		RuntimeType:           joinOptions.RuntimeType,
		InterfaceName:         joinOptions.InterfaceName,
		CertPath:              joinOptions.CertPath,
		RemoteRuntimeEndpoint: joinOptions.RemoteRuntimeEndpoint,
	}

	toolList["MQTT"] = &util.MQTTInstTool{}
	return nil
}

//Execute the installation for each tool and start edgecore
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
