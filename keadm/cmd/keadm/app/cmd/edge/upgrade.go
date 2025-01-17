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
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"k8s.io/klog/v2"
	"sigs.k8s.io/yaml"

	"github.com/kubeedge/api/apis/common/constants"
	"github.com/kubeedge/api/apis/componentconfig/edgecore/v1alpha2"
	api "github.com/kubeedge/api/apis/fsm/v1alpha1"
	"github.com/kubeedge/kubeedge/keadm/cmd/keadm/app/cmd/common"
	"github.com/kubeedge/kubeedge/keadm/cmd/keadm/app/cmd/util"
	"github.com/kubeedge/kubeedge/keadm/cmd/keadm/app/cmd/util/extsystem"
	"github.com/kubeedge/kubeedge/pkg/util/fsm"
)

var (
	// idempotencyRecord is a file that is used to avoid upgrading node twice once a time.
	// If the file exist, we don't allow upgrade node again
	// we only allow upgrade nodes when the file NOT exist
	idempotencyRecord = filepath.Join(util.KubeEdgePath, "idempotency_record")
)

// NewEdgeUpgrade returns KubeEdge edge upgrade command.
func NewEdgeUpgrade() *cobra.Command {
	upgradeOptions := NewUpgradeOptions()

	cmd := &cobra.Command{
		Use:   "edge",
		Short: "Upgrade edge components",
		Long:  "Upgrade edge components. Upgrade the edge node to the desired version.",
		PreRunE: func(_ *cobra.Command, _ []string) error {
			if upgradeOptions.PreRun != "" {
				fmt.Printf("Executing pre-run script: %s\n", upgradeOptions.PreRun)
				if err := util.RunScript(upgradeOptions.PreRun); err != nil {
					return err
				}
			}
			return nil
		},
		RunE: func(_ *cobra.Command, _ []string) error {
			// upgrade edgecore
			return upgradeOptions.Upgrade()
		},
		PostRunE: func(_ *cobra.Command, _ []string) error {
			// post-run script
			if upgradeOptions.PostRun != "" {
				fmt.Printf("Executing post-run script: %s\n", upgradeOptions.PostRun)
				if err := util.RunScript(upgradeOptions.PostRun); err != nil {
					fmt.Printf("Execute post-run script: %s failed: %v\n", upgradeOptions.PostRun, err)
				}
			}
			return nil
		},
	}

	AddUpgradeFlags(cmd, upgradeOptions)
	return cmd
}

// NewUpgradeOptions returns a struct ready for being used for creating cmd join flags.
func NewUpgradeOptions() *UpgradeOptions {
	opts := &UpgradeOptions{}
	opts.ToVersion = "v" + common.DefaultKubeEdgeVersion
	opts.Config = constants.DefaultConfigDir + "edgecore.yaml"

	return opts
}

// Upgrade handles upgrade command logic
func (up *UpgradeOptions) Upgrade() error {
	// get EdgeCore configuration from edgecore.yaml config file
	data, err := os.ReadFile(up.Config)
	if err != nil {
		return fmt.Errorf("failed to read config file %s: %v", up.Config, err)
	}

	configure := &v1alpha2.EdgeCoreConfig{}
	err = yaml.Unmarshal(data, configure)
	if err != nil {
		return fmt.Errorf("failed to unmarshal config file %s: %v", up.Config, err)
	}

	upgrade := Upgrade{
		UpgradeID:      up.UpgradeID,
		HistoryID:      up.HistoryID,
		FromVersion:    up.FromVersion,
		ToVersion:      up.ToVersion,
		TaskType:       up.TaskType,
		Image:          up.Image,
		DisableBackup:  up.DisableBackup,
		ConfigFilePath: up.Config,
		EdgeCoreConfig: configure,
	}

	event := &fsm.Event{
		Type:   "Upgrade",
		Action: api.ActionSuccess,
	}
	defer func() {
		// report upgrade result to cloudhub
		if err = util.ReportTaskResult(configure, upgrade.TaskType, upgrade.UpgradeID, *event); err != nil {
			klog.Errorf("failed to report upgrade result to cloud: %v", err)
		}
		// cleanup idempotency record
		if err = os.Remove(idempotencyRecord); err != nil {
			klog.Errorf("failed to remove idempotency_record file(%s): %v", idempotencyRecord, err)
		}
	}()

	// only allow upgrade when last upgrade finished
	if util.FileExists(idempotencyRecord) {
		event.Action = api.ActionFailure
		event.Msg = "last upgrade not finished, not allowed upgrade again"
		return fmt.Errorf("last upgrade not finished, not allowed upgrade again")
	}

	// create idempotency_record file
	if err := os.MkdirAll(filepath.Dir(idempotencyRecord), 0750); err != nil {
		reason := fmt.Sprintf("failed to create idempotency_record dir: %v", err)
		event.Action = api.ActionFailure
		event.Msg = reason
		return errors.New(reason)
	}
	if _, err := os.Create(idempotencyRecord); err != nil {
		reason := fmt.Sprintf("failed to create idempotency_record file: %v", err)
		event.Action = api.ActionFailure
		event.Msg = reason
		return errors.New(reason)
	}

	// run script to do upgrade operation
	err = upgrade.PreProcess()
	if err != nil {
		event.Action = api.ActionFailure
		event.Msg = fmt.Sprintf("upgrade pre process failed: %v", err)
		return fmt.Errorf("upgrade pre process failed: %v", err)
	}

	err = upgrade.Process()
	if err != nil {
		event.Type = "Rollback"
		rbErr := upgrade.Rollback()
		if rbErr != nil {
			event.Action = api.ActionFailure
			event.Msg = rbErr.Error()
		} else {
			event.Msg = err.Error()
		}
		return fmt.Errorf("upgrade process failed: %v", err)
	}

	return nil
}

func (up *Upgrade) PreProcess() error {
	// download the request version edgecore
	klog.Infof("Begin to download version %s edgecore", up.ToVersion)
	if !up.DisableBackup {
		backupPath := filepath.Join(util.KubeEdgeBackupPath, up.FromVersion)
		if err := os.MkdirAll(backupPath, 0750); err != nil {
			return fmt.Errorf("mkdirall failed: %v", err)
		}

		// backup edgecore.db: copy from origin path to backup path
		if err := copyFile(up.EdgeCoreConfig.DataBase.DataSource, filepath.Join(backupPath, "edgecore.db")); err != nil {
			return fmt.Errorf("failed to backup db: %v", err)
		}
		// backup edgecore.yaml: copy from origin path to backup path
		if err := copyFile(up.ConfigFilePath, filepath.Join(backupPath, "edgecore.yaml")); err != nil {
			return fmt.Errorf("failed to back config: %v", err)
		}
		// backup edgecore: copy from origin path to backup path
		if err := copyFile(filepath.Join(util.KubeEdgeUsrBinPath, util.KubeEdgeBinaryName), filepath.Join(backupPath, util.KubeEdgeBinaryName)); err != nil {
			return fmt.Errorf("failed to backup edgecore: %v", err)
		}
	}

	upgradePath := filepath.Join(util.KubeEdgeUpgradePath, up.ToVersion)
	container, err := util.NewContainerRuntime(up.EdgeCoreConfig.Modules.Edged.TailoredKubeletConfig.ContainerRuntimeEndpoint,
		up.EdgeCoreConfig.Modules.Edged.TailoredKubeletConfig.CgroupDriver)
	if err != nil {
		return fmt.Errorf("failed to new container runtime: %v", err)
	}

	image := up.Image

	err = container.PullImages([]string{image})
	if err != nil {
		return fmt.Errorf("pull image failed: %v", err)
	}
	files := map[string]string{
		filepath.Join(util.KubeEdgeUsrBinPath, util.KubeEdgeBinaryName): filepath.Join(upgradePath, util.KubeEdgeBinaryName),
	}
	err = container.CopyResources(image, files)
	if err != nil {
		return fmt.Errorf("failed to cp file from image to host: %v", err)
	}

	return nil
}

func copyFile(src, dst string) error {
	sourceFileStat, err := os.Stat(src)
	if err != nil {
		return err
	}

	if !sourceFileStat.Mode().IsRegular() {
		return fmt.Errorf("%s is not a regular file", src)
	}

	source, err := os.Open(src)
	if err != nil {
		return err
	}
	defer source.Close()

	// copy file using src file mode
	destination, err := os.OpenFile(dst, os.O_RDWR|os.O_CREATE|os.O_TRUNC, sourceFileStat.Mode())
	if err != nil {
		return err
	}
	defer destination.Close()
	_, err = io.Copy(destination, source)
	return err
}

func (up *Upgrade) Process() error {
	klog.Infof("upgrade process start")

	// stop origin edgecore
	err := util.KillKubeEdgeBinary(util.KubeEdgeBinaryName)
	if err != nil {
		return fmt.Errorf("failed to stop edgecore: %v", err)
	}

	// copy new edgecore from upgradePath to /usr/local/bin
	upgradePath := filepath.Join(util.KubeEdgeUpgradePath, up.ToVersion)
	err = copyFile(filepath.Join(upgradePath, util.KubeEdgeBinaryName), filepath.Join(util.KubeEdgeUsrBinPath, util.KubeEdgeBinaryName))
	if err != nil {
		return fmt.Errorf("failed to cp file: %v", err)
	}

	// set withMqtt to false during upgrading edgecore, it will not affect the MQTT container. This is a temporary workaround and will be modified in v1.15.
	// generate edgecore.service
	if util.HasSystemd() {
		extSystem, err := extsystem.GetExtSystem()
		if err != nil {
			return fmt.Errorf("failed to get ext system, err: %v", err)
		}
		if err := extSystem.ServiceCreate(util.KubeEdgeBinaryName,
			fmt.Sprintf("%s --config %s", filepath.Join(util.KubeEdgeUsrBinPath, util.KubeEdgeBinaryName), up.ConfigFilePath),
			map[string]string{
				constants.DeployMqttContainerEnv: strconv.FormatBool(false),
			},
		); err != nil {
			return fmt.Errorf("failed to create edgecore systemd service, err: %v", err)
		}
	}

	// start new edgecore service
	err = runEdgeCore(false)
	if err != nil {
		return fmt.Errorf("failed to start edgecore: %v", err)
	}

	return nil
}

func (up *Upgrade) Rollback() error {
	return rollback(up.FromVersion, up.EdgeCoreConfig.DataBase.DataSource, up.ConfigFilePath)
}

func rollback(HistoryVersion, dataSource, configFilePath string) error {
	klog.Infof("upgrade rollback process start")

	// stop edgecore
	err := util.KillKubeEdgeBinary(util.KubeEdgeBinaryName)
	if err != nil {
		return fmt.Errorf("failed to stop edgecore: %v", err)
	}

	// rollback origin config/db/binary

	// backup edgecore.db: copy from backup path to origin path
	backupPath := filepath.Join(util.KubeEdgeBackupPath, HistoryVersion)
	if err := copyFile(filepath.Join(backupPath, "edgecore.db"), dataSource); err != nil {
		return fmt.Errorf("failed to rollback db: %v", err)
	}
	// backup edgecore.yaml: copy from backup path to origin path
	if err := copyFile(filepath.Join(backupPath, "edgecore.yaml"), configFilePath); err != nil {
		return fmt.Errorf("failed to back config: %v", err)
	}
	// backup edgecore: copy from backup path to origin path
	if err := copyFile(filepath.Join(backupPath, util.KubeEdgeBinaryName), filepath.Join(util.KubeEdgeUsrBinPath, util.KubeEdgeBinaryName)); err != nil {
		return fmt.Errorf("failed to backup edgecore: %v", err)
	}

	// generate edgecore.service
	if util.HasSystemd() {
		extSystem, err := extsystem.GetExtSystem()
		if err != nil {
			return fmt.Errorf("failed to get ext system, err: %v", err)
		}
		if err := extSystem.ServiceCreate(util.KubeEdgeBinaryName,
			fmt.Sprintf("%s --config %s", filepath.Join(util.KubeEdgeUsrBinPath, util.KubeEdgeBinaryName), configFilePath),
			map[string]string{
				constants.DeployMqttContainerEnv: strconv.FormatBool(false),
			},
		); err != nil {
			return fmt.Errorf("failed to create edgecore systemd service, err: %v", err)
		}
	}

	// start edgecore
	err = runEdgeCore(false)
	if err != nil {
		return fmt.Errorf("failed to start origin edgecore: %v", err)
	}
	return nil
}

func (up *Upgrade) UpdateStatus(status string) {
	up.Status = status
}

func (up *Upgrade) UpdateFailureReason(reason string) {
	up.Reason = reason
}

type UpgradeOptions struct {
	UpgradeID     string
	HistoryID     string
	FromVersion   string
	ToVersion     string
	Config        string
	Image         string
	DisableBackup bool
	TaskType      string
	PreRun        string
	PostRun       string
}

type Upgrade struct {
	UpgradeID      string
	HistoryID      string
	FromVersion    string
	ToVersion      string
	Image          string
	DisableBackup  bool
	ConfigFilePath string
	TaskType       string
	EdgeCoreConfig *v1alpha2.EdgeCoreConfig

	Status string
	Reason string
}

// AddUpgradeFlags adds some flags to the upgrade command, and use UpgradeOptions struct to map these flags.
func AddUpgradeFlags(cmd *cobra.Command, upgradeOptions *UpgradeOptions) {
	cmd.Flags().StringVar(&upgradeOptions.UpgradeID, "upgradeID", upgradeOptions.UpgradeID,
		"Use this key to specify Upgrade CR ID")

	cmd.Flags().StringVar(&upgradeOptions.HistoryID, "historyID", upgradeOptions.HistoryID,
		"Use this key to specify Upgrade CR status history ID.")

	cmd.Flags().StringVar(&upgradeOptions.FromVersion, "fromVersion", upgradeOptions.FromVersion,
		"Use this key to specify the origin version before upgrade")

	cmd.Flags().StringVar(&upgradeOptions.ToVersion, "toVersion", upgradeOptions.ToVersion,
		"Use this key to upgrade the required KubeEdge version")

	cmd.Flags().StringVar(&upgradeOptions.Config, "config", upgradeOptions.Config,
		"Use this key to specify the path to the edgecore configuration file.")

	cmd.Flags().StringVar(&upgradeOptions.Image, "image", upgradeOptions.Image,
		"Use this key to specify installation image to download.")

	cmd.Flags().StringVar(&upgradeOptions.TaskType, "type", "upgrade",
		"Use this key to specify the task type for reporting status.")

	cmd.Flags().BoolVar(&upgradeOptions.DisableBackup, "disable-backup", upgradeOptions.DisableBackup,
		"Use this key to specify the backup enable for upgrade.")

	cmd.Flags().StringVar(&upgradeOptions.PreRun, common.FlagNamePreRun, upgradeOptions.PreRun,
		"Execute the prescript before upgrading the node. (for example: keadm upgrade edge --pre-run=./test-script.sh ...)")

	cmd.Flags().StringVar(&upgradeOptions.PostRun, common.FlagNamePostRun, upgradeOptions.PostRun,
		"Execute the postscript after upgrading the node. (for example: keadm upgrade edge --post-run=./test-script.sh ...)")
}
