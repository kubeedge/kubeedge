package edge

import (
	"fmt"
	"path/filepath"

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
		PreRunE: func(_cmd *cobra.Command, _args []string) error {
			if err := executor.prerun(opts); err != nil {
				// Report results when errors occur in pre-run.
				reporter := upgrdeedge.NewJSONFileReporter(upgrdeedge.EventTypeRollback,
					executor.currentVersion, executor.rollbackTo)
				if reperr := reporter.Report(err); reperr != nil {
					klog.Errorf("failed to report rollback result: %v", reperr)
				}
			}
			return nil
		},
		RunE: func(_cmd *cobra.Command, _args []string) error {
			err := executor.rollback(opts)
			// Report the result of the rollback process.
			reporter := upgrdeedge.NewJSONFileReporter(upgrdeedge.EventTypeRollback,
				executor.currentVersion, executor.rollbackTo)
			if reperr := reporter.Report(err); reperr != nil {
				klog.Errorf("failed to report rollback result: %v", reperr)
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
	AddRollbackFlags(cmd, &opts)
	return cmd
}

type rollbackExecutor struct {
	rollbackTo string

	baseUpgradeExecutor
}

func newRollbackExecutor() rollbackExecutor {
	return rollbackExecutor{baseUpgradeExecutor: baseUpgradeExecutor{}}
}

func (executor *rollbackExecutor) prerun(opts RollbackOptions) error {
	if err := executor.baseUpgradeExecutor.prePrerun(opts.Config); err != nil {
		return err
	}

	// TODO: check HistoricalVersion and set default value if empty
	executor.rollbackTo = opts.HistoricalVersion

	if err := executor.baseUpgradeExecutor.postPrerun(opts.PreRun); err != nil {
		return err
	}
	return nil
}

func (executor *rollbackExecutor) rollback(opts RollbackOptions) error {
	klog.Infof("rollback process start ...")
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
	if err := runEdgeCore(false); err != nil {
		return fmt.Errorf("failed to start edgecore, err: %v", err)
	}
	klog.Infof("rollback process successful")
	return nil
}
