//go:build !windows

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

package edge

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/klog/v2"
	phases "k8s.io/kubernetes/cmd/kubeadm/app/cmd/phases/reset"
	utilsexec "k8s.io/utils/exec"

	"github.com/kubeedge/kubeedge/keadm/cmd/keadm/app/cmd/common"
	"github.com/kubeedge/kubeedge/keadm/cmd/keadm/app/cmd/util"
	"github.com/kubeedge/kubeedge/keadm/cmd/keadm/app/cmd/util/extsystem"
)

var (
	resetLongDescription = `
keadm reset edge command can be executed edge node
In edge node it shuts down the edge processes of KubeEdge
`
	resetExample = `
For edge node edge:
keadm reset edge
`
)

func NewOtherEdgeReset() *cobra.Command {
	const isEdgeNode = true
	reset := util.NewResetOptions()
	step := common.NewStep()
	var cmd = &cobra.Command{
		Use:     "edge",
		Short:   "Teardowns EdgeCore component",
		Long:    resetLongDescription,
		Example: resetExample,
		PreRunE: func(_ *cobra.Command, _ []string) error {
			if reset.PreRun != "" {
				step.Printf("executing pre-run script: %s", reset.PreRun)
				if err := util.RunScript(reset.PreRun); err != nil {
					return err
				}
			}
			return nil
		},
		RunE: func(_ *cobra.Command, _ []string) error {
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

			step.Printf("clean up static pods")
			config, err := util.ParseEdgecoreConfig(common.EdgecoreConfigPath)
			if err != nil {
				klog.Warningf("failed to parse edgecore configuration, skip cleaning up static pods, err: %v", err)
			} else {
				if reset.Endpoint == "" {
					reset.Endpoint = config.Modules.Edged.TailoredKubeletConfig.ContainerRuntimeEndpoint
				}
				staticPodPath := config.Modules.Edged.TailoredKubeletConfig.StaticPodPath
				if staticPodPath != "" {
					if err := phases.CleanDir(staticPodPath); err != nil {
						klog.Warningf("failed to delete static pod directory %s: %v", staticPodPath, err)
					}
				}
			}
			step.Printf("kill edgecore and remove edgecore service")
			if err := TearDownEdgeCore(); err != nil {
				return err
			}
			step.Printf("clean up containers managed by KubeEdge")
			if err := util.RemoveContainers(reset.Endpoint, utilsexec.New()); err != nil {
				klog.Warningf("failed to clean up containers, err: %v", err)
			}
			step.Printf("clean up dirs created by KubeEdge")

			if err := util.CleanDirectories(isEdgeNode); err != nil {
				klog.Warningf("failed to clean up directories, err: %v", err)
			}
			return nil
		},

		PostRun: func(_ *cobra.Command, _ []string) {
			// post-run script
			if reset.PostRun != "" {
				step.Printf("executing post-run script: %s", reset.PostRun)
				if err := util.RunScript(reset.PostRun); err != nil {
					klog.Warningf("execute post-run script: %s failed, err: %v", reset.PostRun, err)
				}
			}
			klog.Info("reset edge node successfully!")
		},
	}

	addResetFlags(cmd, reset)
	return cmd
}

// TearDownEdgeCore will bring down edge component,
func TearDownEdgeCore() error {
	extSystem, err := extsystem.GetExtSystem()
	if err != nil {
		return fmt.Errorf("failed to get init system, err: %v", err)
	}
	service := util.KubeEdgeBinaryName
	if extSystem.ServiceExists(service) {
		klog.V(2).Info("edgecore service is exists")
		if extSystem.ServiceIsActive(service) {
			klog.V(2).Info("edgecore service is active, stopping it")
			if err := extSystem.ServiceStop(service); err != nil {
				klog.Warningf("failed to stop edgecore service, err: %v", err)
			}
			timeout, interval := 10*time.Second, 1*time.Second
			if err := wait.PollUntilContextTimeout(context.Background(), interval, timeout, false,
				func(_ context.Context) (done bool, err error) {
					return !extSystem.ServiceIsActive(service), nil
				},
			); err != nil {
				klog.Warningf("failed to wait for edgecore service to stop, err: %v", err)
			}
		}
		if extSystem.ServiceIsEnabled(service) {
			klog.V(2).Info("edgecore service is enabled, disable it")
			if err := extSystem.ServiceDisable(service); err != nil {
				klog.Warningf("failed to disable edgecore service, err: %v", err)
			}
		}
		klog.V(2).Info("removing edgecore service")
		if err := extSystem.ServiceRemove(service); err != nil {
			klog.Warningf("failed to remove edgecore service, err: %v", err)
		}
	}
	return nil
}

func addResetFlags(cmd *cobra.Command, resetOpts *common.ResetOptions) {
	cmd.Flags().BoolVar(&resetOpts.Force, "force", resetOpts.Force,
		"Reset the node without prompting for confirmation")
	cmd.Flags().StringVar(&resetOpts.Endpoint, "remote-runtime-endpoint", resetOpts.Endpoint,
		"Use this key to set container runtime endpoint")
	cmd.Flags().StringVar(&resetOpts.PreRun, common.FlagNamePreRun, resetOpts.PreRun,
		"Execute the prescript before resetting the node. (for example: keadm reset edge --pre-run=./test-script.sh ...)")
	cmd.Flags().StringVar(&resetOpts.PostRun, common.FlagNamePostRun, resetOpts.PostRun,
		"Execute the postscript after resetting the node. (for example: keadm reset edge --post-run=./test-script.sh ...)")
}
