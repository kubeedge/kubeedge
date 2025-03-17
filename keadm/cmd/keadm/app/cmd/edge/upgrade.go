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
	"bufio"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"k8s.io/klog/v2"

	"github.com/kubeedge/api/apis/common/constants"
	cfgv1alpha2 "github.com/kubeedge/api/apis/componentconfig/edgecore/v1alpha2"
	"github.com/kubeedge/kubeedge/keadm/cmd/keadm/app/cmd/common"
	"github.com/kubeedge/kubeedge/keadm/cmd/keadm/app/cmd/util"
	upgrdeedge "github.com/kubeedge/kubeedge/pkg/upgrade/edge"
	"github.com/kubeedge/kubeedge/pkg/util/files"
)

const upgradeTips = `WARNING: The upgrade command no longer automatically execute backup. 
If backup is required, please interrupt the current upgrade command and execute the backup command 'keadm backup edge'.`

func NewUpgradeCommand() *cobra.Command {
	var opts UpgradeOptions
	executor := newUpgradeExecutor()

	cmd := &cobra.Command{
		Use:   "edge",
		Short: "Upgrade the edge node to the desired version.",
		Long:  "Upgrade the edge node to the desired version.\n" + upgradeTips,
		RunE: func(_cmd *cobra.Command, _args []string) error {
			fmt.Println(upgradeTips)
			// If the opts.UpgradeID is not empty, it means that it‘s the command triggered by the v1alpha1 node upgrade job.
			// At this time, we cannot add input, which will cause the upgrade task to be blocked.
			// The opts.UpgradeID judgment for compatibility with historical versions, It will be removed in v1.23.
			if !opts.Force && opts.UpgradeID == "" {
				fmt.Print("Are you sure you want to proceed? [y/N]: ")
				s := bufio.NewScanner(os.Stdin)
				s.Scan()
				if err := s.Err(); err != nil {
					return err
				}
				if strings.ToLower(s.Text()) != "y" {
					klog.Infof("aborted upgrade operation")
					return nil
				}
			}

			var err error
			defer func() {
				// Report the result of the rollback process.
				reporter := executor.newReporter(opts.UpgradeID, opts.ToVersion)
				if reperr := reporter.Report(err); reperr != nil {
					klog.Errorf("failed to report upgrade result: %v", reperr)
				}
				if err != OccupiedError {
					executor.release()
				}
			}()

			err = executor.prerun(opts)
			if err != nil {
				return err
			}
			err = executor.upgrade(opts)
			if err != nil {
				return err
			}
			executor.runPostRunHook(opts.PostRun)
			return nil
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
	if err := executor.baseUpgradeExecutor.prePreRun(opts.Config); err != nil {
		return err
	}
	if err := executor.baseUpgradeExecutor.postPreRun(opts.PreRun); err != nil {
		return err
	}
	return nil
}

func (executor *upgradeExecutor) upgrade(opts UpgradeOptions) error {
	// Get new edgecore binary from the image.
	klog.Infof("Begin to download %s of edgecore", opts.ToVersion)
	edgecorePath, err := getEdgeCoreBinary(opts, executor.cfg)
	if err != nil {
		return fmt.Errorf("failed to get edgecore binary, err: %v", err)
	}
	defer func() {
		if err := os.RemoveAll(filepath.Dir(edgecorePath)); err != nil {
			klog.Errorf("failed to remove edgecore binary: %v", err)
		}
	}()
	klog.Infof("Upgrade process start ...")
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
	if err := runEdgeCore(); err != nil {
		return fmt.Errorf("failed to start edgecore, err: %v", err)
	}
	klog.Info("Upgrade process successful")
	return nil
}

func (executor *upgradeExecutor) newReporter(upgradeID, toVersion string) upgrdeedge.Reporter {
	var reporter upgrdeedge.Reporter
	if upgradeID != "" {
		// For compatibility with historical versions, It will be removed in v1.23
		reporter = upgrdeedge.NewTaskEventReporter(upgradeID, upgrdeedge.EventTypeUpgrade, executor.cfg)
	} else {
		reporter = upgrdeedge.NewJSONFileReporter(upgrdeedge.EventTypeUpgrade, executor.currentVersion, toVersion)
	}
	return reporter
}

// getEdgeCoreBinary pulls the installation-package image and obtains the edgecore binary from it.
// The edgecore binary is copied to the upgrade path, and the filepath is returned.
func getEdgeCoreBinary(opts UpgradeOptions, config *cfgv1alpha2.EdgeCoreConfig) (string, error) {
	ctx := context.Background()
	container, err := util.NewContainerRuntime(
		config.Modules.Edged.TailoredKubeletConfig.ContainerRuntimeEndpoint,
		config.Modules.Edged.TailoredKubeletConfig.CgroupDriver)
	if err != nil {
		return "", fmt.Errorf("failed to new container runtime, err: %v", err)
	}
	image := opts.Image + ":" + opts.ToVersion
	if err := container.PullImage(ctx, image, nil, nil); err != nil {
		return "", fmt.Errorf("failed to pull image %s, err: %v", image, err)
	}
	containerFilePath := filepath.Join(constants.KubeEdgeUsrBinPath, constants.KubeEdgeBinaryName)
	hostPath := filepath.Join(upgradePath(opts.ToVersion), constants.KubeEdgeBinaryName)
	files := map[string]string{containerFilePath: hostPath}
	if err := container.CopyResources(ctx, image, files); err != nil {
		return "", fmt.Errorf("failed to copy edgecore from %s in the image %s to host %s, err: %v",
			containerFilePath, image, hostPath, err)
	}
	return hostPath, nil
}

// upgradePath returns the path of the upgrade directory.
func upgradePath(ver string) string {
	return filepath.Join(common.KubeEdgeUpgradePath, ver)
}
