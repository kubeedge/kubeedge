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
	"fmt"
	"path/filepath"

	"github.com/spf13/cobra"
	"k8s.io/klog/v2"

	"github.com/kubeedge/api/apis/common/constants"
	"github.com/kubeedge/kubeedge/keadm/cmd/keadm/app/cmd/common"
	upgrdeedge "github.com/kubeedge/kubeedge/pkg/upgrade/edge"
	"github.com/kubeedge/kubeedge/pkg/util/files"
	"github.com/kubeedge/kubeedge/pkg/version"
)

func NewBackupCommand() *cobra.Command {
	var opts BaseOptions
	executor := newBackupExecutor()

	cmd := &cobra.Command{
		Use:   "edge",
		Short: "Back up important files for rollback edgecore.",
		PreRunE: func(_cmd *cobra.Command, _args []string) error {
			if err := executor.prerun(opts); err != nil {
				// Report results when errors occur in pre-run.
				reporter := upgrdeedge.NewJSONFileReporter(upgrdeedge.EventTypeBackup,
					"", executor.currentVersion)
				if reperr := reporter.Report(err); reperr != nil {
					klog.Errorf("failed to report upgrade result: %v", reperr)
				}
			}
			return nil
		},
		RunE: func(_cmd *cobra.Command, _args []string) error {
			err := executor.backup(opts)
			// Report the result of the backup process.
			reporter := upgrdeedge.NewJSONFileReporter(upgrdeedge.EventTypeBackup,
				"", executor.currentVersion)
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
	AddBaseFlags(cmd, &opts)
	return cmd
}

type backupExecutor struct {
	baseUpgradeExecutor
}

func newBackupExecutor() backupExecutor {
	return backupExecutor{baseUpgradeExecutor: baseUpgradeExecutor{}}
}

func (executor *backupExecutor) prerun(opts BaseOptions) error {
	if err := executor.baseUpgradeExecutor.prePrerun(opts.Config); err != nil {
		return err
	}
	if err := executor.baseUpgradeExecutor.postPrerun(opts.PreRun); err != nil {
		return err
	}
	return nil
}

func (executor *backupExecutor) backup(opts BaseOptions) error {
	klog.Infof("backup process start ...")
	backupFiles := []string{
		executor.cfg.DataBase.DataSource,
		opts.Config,
		filepath.Join(constants.KubeEdgeUsrBinPath, constants.KubeEdgeBinaryName),
	}
	backupPath := filepath.Join(common.KubeEdgeBackupPath, version.Get().String())
	for _, file := range backupFiles {
		dest := filepath.Join(backupPath, filepath.Base(file))
		if err := files.FileCopy(file, dest); err != nil {
			return fmt.Errorf("failed to backup file %s, err: %v", file, err)
		}
	}
	klog.Infof("backup process successful")
	return nil
}
