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

package edge

import (
	"fmt"
	"path/filepath"
	"slices"

	"github.com/spf13/cobra"
	"k8s.io/klog/v2"

	"github.com/kubeedge/api/apis/common/constants"
	"github.com/kubeedge/kubeedge/keadm/cmd/keadm/app/cmd/common"
	"github.com/kubeedge/kubeedge/keadm/cmd/keadm/app/cmd/util"
	upgrdeedge "github.com/kubeedge/kubeedge/pkg/upgrade/edge"
	"github.com/kubeedge/kubeedge/pkg/util/files"
)

func NewRollbackCommand() *cobra.Command {
	var opts RollbackOptions
	executor := newRollbackExecutor()

	cmd := &cobra.Command{
		Use:   "edge",
		Short: "Roll back the edge node to the desired version.",
		RunE: func(_cmd *cobra.Command, _args []string) error {
			var err error
			defer func() {
				// Report the result of the rollback process.
				reporter := upgrdeedge.NewJSONFileReporter(upgrdeedge.EventTypeRollback,
					executor.currentVersion, opts.HistoricalVersion)
				if reperr := reporter.Report(err); reperr != nil {
					klog.Errorf("failed to report rollback result: %v", reperr)
				}
				if err != OccupiedError {
					executor.release()
				}
			}()

			err = executor.prerun(&opts)
			if err != nil {
				return err
			}
			err = executor.rollback(opts)
			if err != nil {
				return err
			}
			executor.runPostRunHook(opts.PostRun)
			return nil
		},
	}
	AddRollbackFlags(cmd, &opts)
	return cmd
}

type rollbackExecutor struct {
	baseUpgradeExecutor
}

func newRollbackExecutor() rollbackExecutor {
	return rollbackExecutor{baseUpgradeExecutor: baseUpgradeExecutor{}}
}

func (executor *rollbackExecutor) prerun(opts *RollbackOptions) error {
	if err := executor.baseUpgradeExecutor.prePreRun(opts.Config); err != nil {
		return err
	}

	// If HistoricalVersion is not null, validate it, otherwise set it to the latast version.
	subdirs, err := files.GetSubDirs(common.KubeEdgeBackupPath, true)
	if err != nil {
		return fmt.Errorf("failed to get backup dirs from %s, err: %v",
			common.KubeEdgeBackupPath, err)
	}
	if opts.HistoricalVersion != "" {
		if exist := slices.Contains(subdirs, opts.HistoricalVersion); !exist {
			return fmt.Errorf("the historical version %s is not exist in backup dir %s",
				opts.HistoricalVersion, common.KubeEdgeBackupPath)
		}
	} else {
		if len(subdirs) == 0 {
			return fmt.Errorf("no historical version is found in backup dir %s",
				common.KubeEdgeBackupPath)
		}
		opts.HistoricalVersion = subdirs[0]
	}

	if err := executor.baseUpgradeExecutor.postPreRun(opts.PreRun); err != nil {
		return err
	}
	return nil
}

func (executor *rollbackExecutor) rollback(opts RollbackOptions) error {
	klog.Info("rollback process start ...")
	// Stop origin edgecore.
	if err := util.KillKubeEdgeBinary(constants.KubeEdgeBinaryName); err != nil {
		return fmt.Errorf("failed to stop edgecore, err: %v", err)
	}
	rollbackFilesPathMap := map[string]string{
		"edgecore.db":                executor.cfg.DataBase.DataSource,
		"edgecore.yaml":              opts.Config,
		constants.KubeEdgeBinaryName: filepath.Join(constants.KubeEdgeUsrBinPath, constants.KubeEdgeBinaryName),
	}
	// Rollback backup files.
	backupPath := filepath.Join(common.KubeEdgeBackupPath, opts.HistoricalVersion)
	for backupFile, dest := range rollbackFilesPathMap {
		if err := files.FileCopy(filepath.Join(backupPath, backupFile), dest); err != nil {
			return fmt.Errorf("failed to rollback file %s, err: %v", dest, err)
		}
	}
	// Start new edgecore.
	if err := runEdgeCore(); err != nil {
		return fmt.Errorf("failed to start edgecore, err: %v", err)
	}
	klog.Info("rollback process successful")
	return nil
}
