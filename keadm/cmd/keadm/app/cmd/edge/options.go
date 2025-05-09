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
	"github.com/spf13/cobra"
	"k8s.io/klog/v2"

	"github.com/kubeedge/api/apis/common/constants"
	"github.com/kubeedge/kubeedge/keadm/cmd/keadm/app/cmd/common"
)

type BaseOptions struct {
	// Config is the path to the edgecore config file.
	// The default value is /etc/kubeedge/config/edgecore.yaml.
	Config string
	// PreRun defins the shell file to run before upgrading.
	PreRun string
	// PostRun defins the shell file to run after upgrading.
	PostRun string
}

type UpgradeOptions struct {
	// ToVersion is the version to upgrade to.
	// The default value is the current version of keadm.
	ToVersion string
	// Image uses to specify a custom repository image name of installation-package.
	// The default value is kubeedge/installation-package, the image pulled is <Image>:<ToVersion>
	Image string
	// Force is a flag to force upgrade.
	// If set to true, the upgrade command will not prompt for confirmation.
	Force bool

	BaseOptions

	// UpgradeID is the name of the node upgrade job, used to report upgrade results.
	// Deprecated: using keadm to report upgrade results is not a good way.
	// For compatibility with historical versions, It will be removed in v1.23
	UpgradeID string
	// TaskType is the type of the task.
	// Deprecated: using keadm to report upgrade results is not a good way.
	// For compatibility with historical versions, It will be removed in v1.23
	TaskType string
	// HistoryID a random uuid string.
	// Deprecated: Nowhere to use it.
	// For compatibility with historical versions, It will be removed in v1.23
	HistoryID string
	// FromVersion uses to describe the version before upgrading.
	// Deprecated: It should be obtained by some means rather than manually specified.
	// For compatibility with historical versions, It will be removed in v1.23
	FromVersion string
	// DisableBackup is a flag to disable backup.
	// Deprecated: This field will no longer be valid and a backup command will be provided.
	// For compatibility with historical versions, It will be removed in v1.23
	DisableBackup bool
}

type RollbackOptions struct {
	// HistoricalVersion is the version to roll back to. This version must have been backed up.
	// If not set, get the latest backup version.
	HistoricalVersion string

	BaseOptions
}

// AddBaseFlags adds some common flags to the upgrade related commands, and use BaseOptions struct to map these flags.
func AddBaseFlags(cmd *cobra.Command, opts *BaseOptions) {
	cmd.Flags().StringVar(&opts.Config, "config", constants.EdgecoreConfigPath,
		"Use this key to specify the path to the edgecore configuration file.")
	cmd.Flags().StringVar(&opts.PreRun, common.FlagNamePreRun, opts.PreRun,
		"Execute the prescript before upgrading the node. (for example: keadm upgrade edge --pre-run=./test-script.sh ...)")
	cmd.Flags().StringVar(&opts.PostRun, common.FlagNamePostRun, opts.PostRun,
		"Execute the postscript after upgrading the node. (for example: keadm upgrade edge --post-run=./test-script.sh ...)")
}

// AddUpgradeFlags adds some flags to the upgrade command, and use UpgradeOptions struct to map these flags.
func AddUpgradeFlags(cmd *cobra.Command, opts *UpgradeOptions) {
	AddBaseFlags(cmd, &opts.BaseOptions)

	cmd.Flags().StringVar(&opts.ToVersion, "toVersion", "v"+common.DefaultKubeEdgeVersion,
		"Use this key to upgrade the required KubeEdge version.")
	cmd.Flags().StringVar(&opts.Image, "image", "kubeedge/installation-package",
		"Use this key to specify installation image to download.")
	cmd.Flags().BoolVar(&opts.Force, "force", opts.Force,
		"Upgrade the node without prompting for confirmation")

	// TODO: remove these flags in v1.23
	const deprecatedMessage = "For compatibility with historical versions, It will be removed in v1.23"
	cmd.Flags().StringVar(&opts.UpgradeID, "upgradeID", opts.UpgradeID,
		"Use this key to specify Upgrade CR ID")
	if err := cmd.Flags().MarkDeprecated("upgradeID", deprecatedMessage); err != nil {
		klog.Error(err)
	}
	cmd.Flags().StringVar(&opts.HistoryID, "historyID", opts.HistoryID,
		"Use this key to specify Upgrade CR status history ID.")
	if err := cmd.Flags().MarkDeprecated("historyID", deprecatedMessage); err != nil {
		klog.Error(err)
	}
	cmd.Flags().StringVar(&opts.FromVersion, "fromVersion", opts.FromVersion,
		"Use this key to specify the origin version before upgrade")
	if err := cmd.Flags().MarkDeprecated("fromVersion", deprecatedMessage); err != nil {
		klog.Error(err)
	}
	cmd.Flags().StringVar(&opts.TaskType, "type", "upgrade",
		"Use this key to specify the task type for reporting status.")
	if err := cmd.Flags().MarkDeprecated("type", deprecatedMessage); err != nil {
		klog.Error(err)
	}
	cmd.Flags().BoolVar(&opts.DisableBackup, "disable-backup", opts.DisableBackup,
		"Use this key to specify the backup enable for upgrade.")
	if err := cmd.Flags().MarkDeprecated("disable-backup", deprecatedMessage); err != nil {
		klog.Error(err)
	}
}

// AddRollbackFlags adds some flags to the rollback command, and use RollbackOptions struct to map these flags.
func AddRollbackFlags(cmd *cobra.Command, opts *RollbackOptions) {
	AddBaseFlags(cmd, &opts.BaseOptions)

	cmd.Flags().StringVar(&opts.HistoricalVersion, "historical-version", opts.HistoricalVersion,
		"Use this key to roll back the KubeEdge version, If not set, get the latest backup version.")
}
