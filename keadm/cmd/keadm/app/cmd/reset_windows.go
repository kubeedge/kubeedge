//go:build windows

/*
Copyright 2023 The KubeEdge Authors.

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
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	phases "k8s.io/kubernetes/cmd/kubeadm/app/cmd/phases/reset"
	utilruntime "k8s.io/kubernetes/cmd/kubeadm/app/util/runtime"
	utilsexec "k8s.io/utils/exec"

	"github.com/kubeedge/kubeedge/keadm/cmd/keadm/app/cmd/common"
	"github.com/kubeedge/kubeedge/keadm/cmd/keadm/app/cmd/util"
)

var (
	resetLongDescription = `
keadm reset command in windows can only be executed in edge node.
It shut down the edge processes of KubeEdge.
`
	resetExample = `
keadm reset
`
)

func newResetOptions() *common.ResetOptions {
	opts := &common.ResetOptions{}
	opts.Kubeconfig = common.DefaultKubeConfig
	return opts
}

func NewKubeEdgeReset() *cobra.Command {
	reset := newResetOptions()

	var cmd = &cobra.Command{
		Use:     "reset",
		Short:   "Teardowns KubeEdge edge component in windows server",
		Long:    resetLongDescription,
		Example: resetExample,
		PreRunE: func(cmd *cobra.Command, args []string) error {
			if !util.IsNSSMInstalled() {
				fmt.Println("Seems like you haven't exec 'keadm join' in this host, because nssm not found in system path (auto installed by 'keadm join'), exit")
				os.Exit(0)
			}
			whoRunning := util.RunningModuleV2(reset)
			if whoRunning == common.NoneRunning && !reset.Force {
				fmt.Println("Edgecore service installed by nssm not found in this host, exit. If you want to clean the related files, using flag --force")
				os.Exit(0)
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			if !reset.Force {
				fmt.Println("[reset] WARNING: Changes made to this host by 'keadm join' will be reverted.")
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
			// 1. kill edgecore process.
			// For edgecore, don't delete node from K8S
			if err := TearDownKubeEdge(reset.Kubeconfig); err != nil {
				err = fmt.Errorf("err when stop and remove edgecore using nssm: %s", err.Error())
				fmt.Print("[reset] No edgecore running now, do you want to clean all the related directories? [y/N]: ")
				s := bufio.NewScanner(os.Stdin)
				s.Scan()
				if err := s.Err(); err != nil {
					return err
				}
				if strings.ToLower(s.Text()) != "y" {
					return fmt.Errorf("aborted reset operation")
				}
				return cleanDirectories()
			}

			// 2. Remove containers managed by KubeEdge.
			if err := RemoveContainers(utilsexec.New()); err != nil {
				fmt.Printf("Failed to remove containers: %v\n", err)
			}

			// 3. Clean stateful directories
			if err := cleanDirectories(); err != nil {
				return err
			}

			fmt.Println("Reset Complete")

			return nil
		},
	}

	addResetFlags(cmd, reset)
	return cmd
}

// TearDownKubeEdge will bring down edge components,
// depending upon in which type of node it is executed
func TearDownKubeEdge(_ string) error {
	// 1.1 stop check if running now, stop it if running
	if util.IsNSSMServiceRunning(util.KubeEdgeBinaryName) {
		fmt.Println("Egdecore service is running, stop...")
		if _err := util.StopNSSMService(util.KubeEdgeBinaryName); _err != nil {
			return _err
		}
		fmt.Println("Egdecore service stop success.")
	}

	// 1.2 remove nssm service
	fmt.Println("Start removing egdecore service using nssm")
	_err := util.UninstallNSSMService(util.KubeEdgeBinaryName)
	if _err != nil {
		return _err
	}
	fmt.Println("Egdecore service remove complete")
	return nil
}

// RemoveContainers removes all Kubernetes-managed containers
func RemoveContainers(execer utilsexec.Interface) error {
	fmt.Println("Start removing containers managed by KubeEdge")
	criSocketPath, err := utilruntime.DetectCRISocket()
	if err != nil {
		return err
	}

	containerRuntime, err := utilruntime.NewContainerRuntime(execer, criSocketPath)
	if err != nil {
		return err
	}

	containers, err := containerRuntime.ListKubeContainers()
	if err != nil {
		return err
	}

	err = containerRuntime.RemoveContainers(containers)
	if err != nil {
		return err
	}
	fmt.Println("Rremoving containers success")
	return nil
}

func cleanDirectories() error {
	fmt.Println("Start cleaning directories...")
	var dirToClean = []string{
		util.KubeEdgePath,
		util.KubeEdgeLogPath,
		util.KubeEdgeSocketPath,
		util.EdgeRootDir,
		util.KubeEdgeUsrBinPath,
	}

	for _, dir := range dirToClean {
		if err := phases.CleanDir(dir); err != nil {
			fmt.Printf("Failed to delete directory %s: %v\n", dir, err)
		}
	}

	fmt.Println("Cleaning directories complete")
	return nil
}

func addResetFlags(cmd *cobra.Command, resetOpts *common.ResetOptions) {
	cmd.Flags().BoolVar(&resetOpts.Force, "force", resetOpts.Force,
		"Reset the node without prompting for confirmation, and continue even if running edgecore not found")
	cmd.Flags().StringVar(&resetOpts.Endpoint, common.FlagNameRemoteRuntimeEndpoint, resetOpts.Endpoint,
		"Use this key to set container runtime endpoint")
}
