/*
Copyright 2025 The KubeEdge Authors.

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
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"k8s.io/klog/v2"

	"github.com/kubeedge/api/apis/common/constants"
	edgeconfig "github.com/kubeedge/api/apis/componentconfig/edgecore/v1alpha2"
	"github.com/kubeedge/kubeedge/keadm/cmd/keadm/app/cmd/common"
	"github.com/kubeedge/kubeedge/keadm/cmd/keadm/app/cmd/util"
	upgrdeedge "github.com/kubeedge/kubeedge/pkg/upgrade/edge"
	"github.com/kubeedge/kubeedge/pkg/util/files"
)

const upgradeTips = `[upgrade] WARNING: The upgrade command no longer automatically execute backup. 
[upgrade] If backup is required, please interrupt the current upgrade command and execute the backup command 'keadm backup edge'.
[upgrade] Do you want to continue? [y/N]: `

func NewUpgradeCommand() *cobra.Command {
	var opts UpgradeOptions
	executor := newUpgradeExecutor()

	cmd := &cobra.Command{
		Use:   "edge",
		Short: "Upgrade the edge node to the desired version.",
		Long:  "Upgrade the edge node to the desired version.\n" + upgradeTips,
		PreRunE: func(_cmd *cobra.Command, _args []string) error {
			if !opts.Force {
				fmt.Print(upgradeTips)
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

			if err := executor.prerun(opts); err != nil {
				// Report results when errors occur in pre-run.
				reporter := executor.newReporter(opts.UpgradeID, opts.ToVersion)
				if reperr := reporter.Report(err); reperr != nil {
					klog.Errorf("failed to report upgrade result: %v", reperr)
				}
			}
			return nil
		},
		RunE: func(_cmd *cobra.Command, _args []string) error {
			err := executor.upgrade(opts)
			// Report the result of the upgrade process.
			reporter := executor.newReporter(opts.UpgradeID, opts.ToVersion)
			if reperr := reporter.Report(err); reperr != nil {
				klog.Errorf("failed to report upgrade result: %v", reperr)
			}
			return err
		},
		PostRun: func(_cmd *cobra.Command, _args []string) {
			defer func() {
				executor.release()
			}()
			executor.runPostrunHook(opts.PostRun)
		},
	}
	AddUpgradeFlags(cmd, &opts)
	return cmd
}

type upgradeExecutor struct {
	baseUpgradeExecutor
}

func newUpgradeExecutor() upgradeExecutor {
	return upgradeExecutor{baseUpgradeExecutor: baseUpgradeExecutor{}}
}

func (executor *upgradeExecutor) prerun(opts UpgradeOptions) error {
	if err := executor.baseUpgradeExecutor.prePrerun(opts.Config); err != nil {
		return err
	}
	if err := executor.baseUpgradeExecutor.postPrerun(opts.PreRun); err != nil {
		return err
	}
	return nil
}

func (executor *upgradeExecutor) upgrade(opts UpgradeOptions) error {
	// Get new edgecore binary from the image.
	klog.Infof("begin to download %s of edgecore", opts.ToVersion)
	edgecorePath, err := getEdgeCoreBinary(opts, executor.cfg)
	if err != nil {
		return fmt.Errorf("failed to get edgecore binary, err: %v", err)
	}
	klog.Infof("upgrade process start ...")
	// Stop origin edgecore.
	if err := util.KillKubeEdgeBinary(constants.KubeEdgeBinaryName); err != nil {
		return fmt.Errorf("failed to stop edgecore, err: %v", err)
	}
	// Copy new edgecore to /usr/local/bin.
	dest := filepath.Join(constants.KubeEdgeUsrBinPath, constants.KubeEdgeBinaryName)
	if err := files.FileCopy(edgecorePath, dest); err != nil {
		return fmt.Errorf("failed to copy edgecore to %s, err: %v", dest, err)
	}
	// Start new edgecore.
	if err := runEdgeCore(false); err != nil {
		return fmt.Errorf("failed to start edgecore, err: %v", err)
	}
	klog.Info("upgrade process successful")
	return nil
}

func (executor *upgradeExecutor) newReporter(upgradeID, toVersion string) upgrdeedge.Reporter {
	var reporter upgrdeedge.Reporter
	if upgradeID != "" {
		// TODO: For compatibility with historical versions, It will be removed in v1.23
		reporter = upgrdeedge.NewTaskEventReporter(upgradeID, upgrdeedge.EventTypeUpgrade, executor.cfg)
	} else {
		// TODO: get edgecore version
		reporter = upgrdeedge.NewJSONFileReporter(upgrdeedge.EventTypeUpgrade, executor.currentVersion, toVersion)
	}
	return reporter
}

// getEdgeCoreBinary pulls the installation-package image and obtains the edgecore binary from it.
// The edgecore binary is copied to the upgrade path, and the filepath is returned.
func getEdgeCoreBinary(opts UpgradeOptions, config *edgeconfig.EdgeCoreConfig) (string, error) {
	container, err := util.NewContainerRuntime(
		config.Modules.Edged.TailoredKubeletConfig.ContainerRuntimeEndpoint,
		config.Modules.Edged.TailoredKubeletConfig.CgroupDriver)
	if err != nil {
		return "", fmt.Errorf("failed to new container runtime, err: %v", err)
	}
	image := opts.Image + ":" + opts.ToVersion
	if err := container.PullImage(image, nil, nil); err != nil {
		return "", fmt.Errorf("failed to pull image %s, err: %v", image, err)
	}
	containerFilePath := filepath.Join(constants.KubeEdgeUsrBinPath, constants.KubeEdgeBinaryName)
	hostPath := filepath.Join(upgradePath(opts.ToVersion), constants.KubeEdgeBinaryName)
	files := map[string]string{containerFilePath: hostPath}
	if err := container.CopyResources(image, files); err != nil {
		return "", fmt.Errorf("failed to copy edgecore from %s in the image %s to host %s, err: %v",
			containerFilePath, image, hostPath, err)
	}
	return hostPath, nil
}

// upgradePath returns the path of the upgrade directory.
func upgradePath(ver string) string {
	return filepath.Join(common.KubeEdgeUpgradePath, ver)
}
