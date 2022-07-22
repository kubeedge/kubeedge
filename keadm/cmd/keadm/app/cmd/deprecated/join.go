/*
Copyright 2022 The KubeEdge Authors.

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

package deprecated

import (
	"fmt"
	"strings"

	"github.com/blang/semver"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	"github.com/kubeedge/kubeedge/common/constants"
	types "github.com/kubeedge/kubeedge/keadm/cmd/keadm/app/cmd/common"
	"github.com/kubeedge/kubeedge/keadm/cmd/keadm/app/cmd/edge"
	"github.com/kubeedge/kubeedge/keadm/cmd/keadm/app/cmd/util"
)

var (
	edgeJoinLongDescription = `
Deprecated: 
"keadm deprecated join" command bootstraps KubeEdge's worker node (at the edge) component.
It will also connect with cloud component to receive
further instructions and forward telemetry data from
devices to cloud
`
	edgeJoinExample = `
Deprecated:
keadm deprecated join --cloudcore-ipport=<ip:port address> --edgenode-name=<unique string as edge identifier>

  - For this command --cloudcore-ipport flag is a required option
  - This command will download and install the default version of pre-requisites and KubeEdge

keadm deprecated join --cloudcore-ipport=10.20.30.40:10000 --edgenode-name=testing123 --kubeedge-version=%s
`
)

// NewDeprecatedEdgeJoin returns KubeEdge edge join command.
func NewDeprecatedEdgeJoin() *cobra.Command {
	joinOptions := newJoinOptions()

	tools := make(map[string]types.ToolsInstaller)
	flagVals := make(map[string]types.FlagData)

	cmd := &cobra.Command{
		Use:     "join",
		Short:   "Deprecated: Bootstraps edge component. Checks and install (if required) the pre-requisites. Execute it on any edge node machine you wish to join",
		Long:    edgeJoinLongDescription,
		Example: fmt.Sprintf(edgeJoinExample, types.DefaultKubeEdgeVersion),
		RunE: func(cmd *cobra.Command, args []string) error {
			//Visit all the flags and store their values and default values.
			checkFlags := func(f *pflag.Flag) {
				util.AddToolVals(f, flagVals)
			}
			cmd.Flags().VisitAll(checkFlags)

			err := Add2EdgeToolsList(tools, flagVals, joinOptions)
			if err != nil {
				return err
			}
			return executeEdge(tools)
		},
	}

	edge.AddJoinOtherFlags(cmd, joinOptions)
	return cmd
}

// newJoinOptions returns a struct ready for being used for creating cmd join flags.
func newJoinOptions() *types.JoinOptions {
	opts := &types.JoinOptions{}
	opts.CertPath = types.DefaultCertPath

	return opts
}

//Add2EdgeToolsList Reads the flagData (containing val and default val) and join options to fill the list of tools.
func Add2EdgeToolsList(toolList map[string]types.ToolsInstaller, flagData map[string]types.FlagData, joinOptions *types.JoinOptions) error {
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

// executeEdge executes the installation for each tool and start edgecore
func executeEdge(toolList map[string]types.ToolsInstaller) error {
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
