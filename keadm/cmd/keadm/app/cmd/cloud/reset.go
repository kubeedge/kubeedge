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

package cloud

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/kubeedge/kubeedge/keadm/cmd/keadm/app/cmd/common"
	"github.com/kubeedge/kubeedge/keadm/cmd/keadm/app/cmd/helm"
	"github.com/kubeedge/kubeedge/keadm/cmd/keadm/app/cmd/util"
)

var (
	resetLongDescription = `
keadm reset cloud command can be executed edge node
In cloud node it shuts down the cloud processes of KubeEdge
`
	resetExample = `
For cloud node:
keadm reset cloud
`
)

func NewCloudReset() *cobra.Command {
	isEdgeNode := false
	reset := util.NewResetOptions()

	var cmd = &cobra.Command{
		Use:     "cloud",
		Short:   "Teardowns CloudCore component",
		Long:    resetLongDescription,
		Example: resetExample,
		PreRunE: func(cmd *cobra.Command, args []string) error {
			whoRunning := util.CloudCoreRunningModuleV2(reset)
			if whoRunning == common.NoneRunning {
				fmt.Println("None of CloudCore components are running in this host, exit")
				os.Exit(0)
			}

			if whoRunning == common.KubeEdgeEdgeRunning {
				isEdgeNode = true
			}

			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			if !reset.Force {
				fmt.Println("[reset] WARNING: Changes made to this host by 'keadm init' or 'keadm join' will be reverted.")
				fmt.Print("[reset] Are you sure you want to proceed? [y/N]: ")
				s := bufio.NewScanner(os.Stdin)
				s.Scan()
				if err := s.Err(); err != nil {
					return err
				}
				if strings.ToLower(s.Text()) != "y" {
					return fmt.Errorf("aborted reset operation")
				}
			}

			// 1. kill cloudcore process.
			if err := TearDownCloudCore(reset.Kubeconfig); err != nil {
				return err
			}

			// 3. Clean stateful directories
			if err := util.CleanDirectories(isEdgeNode); err != nil {
				return err
			}

			//4. TODO: clean status information

			return nil
		},
	}

	addResetFlags(cmd, reset)
	return cmd
}

// TearDownCloudCore will bring down cloud component,
func TearDownCloudCore(kubeConfig string) error {
	ke := &helm.KubeCloudHelmInstTool{
		Common: util.Common{
			KubeConfig: kubeConfig,
		},
	}

	err := ke.TearDown()
	if err != nil {
		return fmt.Errorf("TearDown failed, err:%v", err)
	}
	return nil
}

func addResetFlags(cmd *cobra.Command, resetOpts *common.ResetOptions) {
	cmd.Flags().StringVar(&resetOpts.Kubeconfig, common.FlagNameKubeConfig, common.DefaultKubeConfig,
		"Use this key to set kube-config path, eg: $HOME/.kube/config")
	cmd.Flags().BoolVar(&resetOpts.Force, "force", resetOpts.Force,
		"Reset the node without prompting for confirmation")
}
