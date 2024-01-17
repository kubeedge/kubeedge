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

package edge

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/kubeedge/kubeedge/common/constants"
	"github.com/kubeedge/kubeedge/keadm/cmd/keadm/app/cmd/common"
	"github.com/kubeedge/kubeedge/keadm/cmd/keadm/app/cmd/util"
	"github.com/kubeedge/kubeedge/pkg/apis/componentconfig/edgecore/v1alpha2"
	"github.com/kubeedge/viaduct/pkg/api"
)

var (
	edgeJoinDescription = `
"keadm join" command bootstraps KubeEdge's worker node (at the edge) component.
It will also connect with cloud component to receive
further instructions and forward telemetry data from
devices to cloud
`
	edgeJoinExample = `
keadm join --cloudcore-ipport=<ip:port address> --edgenode-name=<unique string as edge identifier>

  - For this command --cloudcore-ipport flag is a required option
  - This command will download and install the default version of pre-requisites and KubeEdge

keadm join --cloudcore-ipport=10.20.30.40:10000 --edgenode-name=testing123 --kubeedge-version=v` + common.DefaultKubeEdgeVersion + `
`
)

var edgeCoreConfig *v1alpha2.EdgeCoreConfig

func NewEdgeJoin() *cobra.Command {
	joinOptions := newOption()
	step := common.NewStep()
	cmd := &cobra.Command{
		Use:          "join",
		Short:        "Bootstraps edge component. Checks and install (if required) the pre-requisites. Execute it on any edge node machine you wish to join",
		Long:         edgeJoinDescription,
		Example:      edgeJoinExample,
		SilenceUsage: true,
		PreRunE: func(cmd *cobra.Command, args []string) error {
			step.Printf("Check KubeEdge edgecore process status")
			running, err := util.IsKubeEdgeProcessRunning(util.KubeEdgeBinaryName)
			if err != nil {
				return fmt.Errorf("check KubeEdge edgecore process status failed: %v", err)
			}
			if running {
				return fmt.Errorf("EdgeCore is already running on this node, please run reset to clean up first")
			}

			step.Printf("Check if the management directory is clean")
			if _, err := os.Stat(util.KubeEdgePath); err != nil {
				if os.IsNotExist(err) {
					return nil
				}
				return fmt.Errorf("Stat management directory %s failed: %v", util.KubeEdgePath, err)
			}
			entries, err := os.ReadDir(util.KubeEdgePath)
			if err != nil {
				return fmt.Errorf("read management directory %s failed: %v", util.KubeEdgePath, err)
			}
			if len(entries) > 0 {
				return fmt.Errorf("the management directory %s is not clean, please remove it first", util.KubeEdgePath)
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			ver, err := util.GetCurrentVersion(joinOptions.KubeEdgeVersion)
			if err != nil {
				return fmt.Errorf("edge node join failed: %v", err)
			}
			joinOptions.KubeEdgeVersion = ver

			if err := join(joinOptions, step); err != nil {
				return fmt.Errorf("edge node join failed: %v", err)
			}

			return nil
		},
	}

	AddJoinOtherFlags(cmd, joinOptions)
	return cmd
}

func newOption() *common.JoinOptions {
	joinOptions := &common.JoinOptions{}
	joinOptions.WithMQTT = false
	joinOptions.CGroupDriver = v1alpha2.CGroupDriverCGroupFS
	joinOptions.CertPath = common.DefaultCertPath
	joinOptions.RemoteRuntimeEndpoint = constants.DefaultRemoteRuntimeEndpoint
	joinOptions.HubProtocol = api.ProtocolTypeWS
	return joinOptions
}

func createDirs() error {
	// Create management directory
	if err := os.MkdirAll(util.KubeEdgePath, os.ModePerm); err != nil {
		return fmt.Errorf("create %s folder path failed: %v", util.KubeEdgePath, err)
	}
	// Create config directory
	if err := os.MkdirAll(util.KubeEdgeConfigDir, os.ModePerm); err != nil {
		return fmt.Errorf("create %s folder path failed: %v", util.KubeEdgeConfigDir, err)
	}
	// Create log directory
	if err := os.MkdirAll(util.KubeEdgeLogPath, os.ModePerm); err != nil {
		return fmt.Errorf("create %s folder path failed: %v", util.KubeEdgeLogPath, err)
	}
	// Create resource directory
	if err := os.MkdirAll(util.KubeEdgeSocketPath, os.ModePerm); err != nil {
		return fmt.Errorf("create %s folder path failed: %v", util.KubeEdgeSocketPath, err)
	}
	return nil
}

func setEdgedNodeLabels(opt *common.JoinOptions) map[string]string {
	labelsMap := make(map[string]string)
	for _, label := range opt.Labels {
		arr := strings.SplitN(label, "=", 2)
		if arr[0] == "" {
			continue
		}

		if len(arr) > 1 {
			labelsMap[arr[0]] = arr[1]
		} else {
			labelsMap[arr[0]] = ""
		}
	}
	return labelsMap
}

func createBootstrapFile(opt *common.JoinOptions) error {
	bootstrapFile := constants.BootstrapFile
	_, err := os.Create(bootstrapFile)
	if err != nil {
		return err
	}

	// write token to bootstrap-edgecore.conf file
	token := []byte(opt.Token)
	return os.WriteFile(bootstrapFile, token, 0640)
}
