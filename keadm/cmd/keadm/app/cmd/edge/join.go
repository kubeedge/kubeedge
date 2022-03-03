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
	"strings"

	"github.com/blang/semver"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	"github.com/kubeedge/kubeedge/common/constants"
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

keadm join --cloudcore-ipport=10.20.30.40:10000 --edgenode-name=testing123 --kubeedge-version=%s
`
)

// NewEdgeJoin returns KubeEdge edge join command.
func NewEdgeJoin() *cobra.Command {
	joinOptions := newJoinOptions()

	tools := make(map[string]types.ToolsInstaller)
	flagVals := make(map[string]types.FlagData)

	cmd := &cobra.Command{
		Use:     "join",
		Short:   "Bootstraps edge component. Checks and install (if required) the pre-requisites. Execute it on any edge node machine you wish to join",
		Long:    edgeJoinLongDescription,
		Example: fmt.Sprintf(edgeJoinExample, types.DefaultKubeEdgeVersion),
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

	cmd.Flags().StringVar(&joinOptions.CGroupDriver, types.CGroupDriver, joinOptions.CGroupDriver,
		"CGroupDriver that uses to manipulate cgroups on the host (cgroupfs or systemd), the default value is cgroupfs")

	cmd.Flags().StringVar(&joinOptions.CertPath, types.CertPath, joinOptions.CertPath,
		fmt.Sprintf("The certPath used by edgecore, the default value is %s", types.DefaultCertPath))

	cmd.Flags().StringVarP(&joinOptions.CloudCoreIPPort, types.CloudCoreIPPort, "e", joinOptions.CloudCoreIPPort,
		"IP:Port address of KubeEdge CloudCore")

	if err := cmd.MarkFlagRequired(types.CloudCoreIPPort); err != nil {
		fmt.Printf("mark flag required failed with error: %v\n", err)
	}

	cmd.Flags().StringVarP(&joinOptions.RuntimeType, types.RuntimeType, "r", joinOptions.RuntimeType,
		"Container runtime type")

	cmd.Flags().StringVarP(&joinOptions.EdgeNodeName, types.EdgeNodeName, "i", joinOptions.EdgeNodeName,
		"KubeEdge Node unique identification string, If flag not used then the command will generate a unique id on its own")

	cmd.Flags().StringVarP(&joinOptions.RemoteRuntimeEndpoint, types.RemoteRuntimeEndpoint, "p", joinOptions.RemoteRuntimeEndpoint,
		"KubeEdge Edge Node RemoteRuntimeEndpoint string, If flag not set, it will use unix:///var/run/dockershim.sock")

	cmd.Flags().StringVarP(&joinOptions.Token, types.Token, "t", joinOptions.Token,
		"Used for edge to apply for the certificate")

	cmd.Flags().StringVarP(&joinOptions.CertPort, types.CertPort, "s", joinOptions.CertPort,
		"The port where to apply for the edge certificate")

	cmd.Flags().StringVar(&joinOptions.TarballPath, types.TarballPath, joinOptions.TarballPath,
		"Use this key to set the temp directory path for KubeEdge tarball, if not exist, download it")

	cmd.Flags().StringSliceVarP(&joinOptions.Labels, types.Labels, "l", joinOptions.Labels,
		`use this key to set the customized labels for node. you can input customized labels like key1=value1,key2=value2`)

	cmd.Flags().BoolVar(&joinOptions.WithMQTT, "with-mqtt", joinOptions.WithMQTT,
		`use this key to set whether to install and start MQTT Broker by default`)
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
			version, err := util.GetLatestVersion()
			if err != nil {
				fmt.Println("Failed to get the latest KubeEdge release version")
				continue
			}
			if len(version) > 0 {
				kubeVer = strings.TrimPrefix(version, "v")
				latestVersion = version
				break
			}
		}
		if len(latestVersion) == 0 {
			fmt.Println("Failed to get the latest KubeEdge release version, will use default version")
			kubeVer = types.DefaultKubeEdgeVersion
		}
	}
	toolList[constants.ProjectName] = &util.KubeEdgeInstTool{
		Common: util.Common{
			ToolVersion: semver.MustParse(kubeVer),
		},
		CloudCoreIP:           joinOptions.CloudCoreIPPort,
		EdgeNodeName:          joinOptions.EdgeNodeName,
		RuntimeType:           joinOptions.RuntimeType,
		CertPath:              joinOptions.CertPath,
		RemoteRuntimeEndpoint: joinOptions.RemoteRuntimeEndpoint,
		Token:                 joinOptions.Token,
		CertPort:              joinOptions.CertPort,
		CGroupDriver:          joinOptions.CGroupDriver,
		TarballPath:           joinOptions.TarballPath,
		Labels:                joinOptions.Labels,
	}

	toolList["MQTT"] = &util.MQTTInstTool{}
	return nil
}

//Execute the installation for each tool and start edgecore
func Execute(toolList map[string]types.ToolsInstaller) error {
	//Install all the required pre-requisite tools
	for name, tool := range toolList {
		if name != constants.ProjectName {
			err := tool.InstallTools()
			if err != nil {
				return err
			}
		}
	}

	//Install and Start KubeEdge Node
	return toolList[constants.ProjectName].InstallTools()
}
