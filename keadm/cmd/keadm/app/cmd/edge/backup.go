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
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"k8s.io/klog/v2"

	"github.com/kubeedge/api/apis/common/constants"
	"github.com/kubeedge/kubeedge/keadm/cmd/keadm/app/cmd/common"
	upgrdeedge "github.com/kubeedge/kubeedge/pkg/upgrade/edge"
	"github.com/kubeedge/kubeedge/pkg/util/files"
)

func NewBackupCommand() *cobra.Command {
	var opts BaseOptions
	executor := newBackupExecutor()

	cmd := &cobra.Command{
		Use:   "edge",
		Short: "Back up important files for rollback edgecore.",
		RunE: func(_cmd *cobra.Command, _args []string) error {
			var err error
			defer func() {
				// Report the result of the backup process.
				reporter := upgrdeedge.NewJSONFileReporter(upgrdeedge.EventTypeBackup,
					"", executor.currentVersion)
				if reperr := reporter.Report(err); reperr != nil {
					klog.Errorf("failed to report backup result: %v", reperr)
				}
				if err != OccupiedError {
					executor.release()
				}
			}()
			err = executor.prerun(opts)
			if err != nil {
				return err
			}
			err = executor.backup(opts)
			if err != nil {
				return err
			}
			executor.runPostRunHook(opts.PostRun)
			return nil
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
	if err := executor.baseUpgradeExecutor.prePreRun(opts.Config); err != nil {
		return err
	}
	if executor.currentVersion == "" ||
		executor.currentVersion == unknownEdgeCoreVersion {
		return errors.New("cannot get the required current version")
	}
	if err := executor.baseUpgradeExecutor.postPreRun(opts.PreRun); err != nil {
		return err
	}
	return nil
}

func (executor *backupExecutor) backup(opts BaseOptions) error {
	klog.Info("backup process start ...")
	backupFiles := []string{
		executor.cfg.DataBase.DataSource,
		opts.Config,
		filepath.Join(constants.KubeEdgeUsrBinPath, constants.KubeEdgeBinaryName),
	}
	backupPath := filepath.Join(common.KubeEdgeBackupPath, executor.currentVersion)
	if err := os.MkdirAll(backupPath, os.ModePerm); err != nil {
		return fmt.Errorf("failed to create backup path %s, err: %v", backupPath, err)
	}
	for _, file := range backupFiles {
		dest := filepath.Join(backupPath, filepath.Base(file))
		if err := files.FileCopy(file, dest); err != nil {
			return fmt.Errorf("failed to backup file %s, err: %v", file, err)
		}
	}
	klog.Info("backup process successful")
	return nil
}
