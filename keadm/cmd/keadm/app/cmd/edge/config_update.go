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
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"k8s.io/klog/v2"
	"sigs.k8s.io/yaml"

	"github.com/kubeedge/api/apis/common/constants"
	"github.com/kubeedge/api/apis/componentconfig/edgecore/v1alpha2"
	api "github.com/kubeedge/api/apis/fsm/v1alpha1"
	"github.com/kubeedge/kubeedge/keadm/cmd/keadm/app/cmd/common"
	"github.com/kubeedge/kubeedge/keadm/cmd/keadm/app/cmd/util"
	"github.com/kubeedge/kubeedge/pkg/util/fsm"
	"github.com/kubeedge/kubeedge/pkg/version"
)

// NewEdgeConfigUpdate returns KubeEdge edge config update command.
func NewEdgeConfigUpdate() *cobra.Command {
	updateOptions := NewConfigUpdateOptions()

	cmd := &cobra.Command{
		Use:   "config-update",
		Short: "Update EdgeCore configuration.",
		Long:  "Update EdgeCore configuration.",
		PreRunE: func(cmd *cobra.Command, args []string) error {
			if updateOptions.PreRun != "" {
				fmt.Printf("Executing pre-run script: %s\n", updateOptions.PreRun)
				if err := util.RunScript(updateOptions.PreRun); err != nil {
					return err
				}
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			// update configuration
			return updateOptions.ConfigUpdate()
		},
		PostRunE: func(cmd *cobra.Command, args []string) error {
			// post-run script
			if updateOptions.PostRun != "" {
				fmt.Printf("Executing post-run script: %s\n", updateOptions.PostRun)
				if err := util.RunScript(updateOptions.PostRun); err != nil {
					fmt.Printf("Execute post-run script: %s failed: %v\n", updateOptions.PostRun, err)
				}
			}
			return nil
		},
	}

	AddConfigUpdateFlags(cmd, updateOptions)
	return cmd
}

// AddConfigUpdateFlags adds some flags to the config-update command, and use ConfigUpdateOptions struct to map these flags.
func AddConfigUpdateFlags(cmd *cobra.Command, updateOptions *ConfigUpdateOptions) {
	cmd.Flags().StringVar(&updateOptions.UpdateID, "updateID", updateOptions.UpdateID,
		"Use this key to specify Upgrade CR ID")

	cmd.Flags().StringVar(&updateOptions.Sets, common.FlagNameSet, updateOptions.Sets,
		`Set values on the command line (can specify multiple or separate values with commas: key1=val1,key2=val2)`)

	cmd.Flags().StringVar(&updateOptions.ConfigPath, "configPath", updateOptions.ConfigPath,
		"Use this key to specify the path to the edgecore configuration file.")

	cmd.Flags().StringVar(&updateOptions.NodeVersion, "nodeVersion", updateOptions.NodeVersion,
		"Use this key to specify the edgecore current version.")
}

func NewConfigUpdateOptions() *ConfigUpdateOptions {
	opts := &ConfigUpdateOptions{}
	opts.ConfigPath = constants.DefaultConfigDir + "edgecore.yaml"
	opts.NodeVersion = version.Get().GitVersion

	return opts
}

// ConfigUpdate handles config-update command logic
func (up *ConfigUpdateOptions) ConfigUpdate() error {
	// get EdgeCore configuration from edgecore.yaml config file
	data, err := os.ReadFile(up.ConfigPath)
	if err != nil {
		return fmt.Errorf("failed to read config file %s: %v", up.Config, err)
	}

	edgeConfigure := &v1alpha2.EdgeCoreConfig{}
	err = yaml.Unmarshal(data, edgeConfigure)
	if err != nil {
		return fmt.Errorf("failed to unmarshal config file %s: %v", up.Config, err)
	}
	up.OldConfig = edgeConfigure

	err = util.ParseSet(edgeConfigure, up.Sets)
	if err != nil {
		return fmt.Errorf("failed to parse config %s: %v", up.Sets, err)
	}

	configUpdate := ConfigUpdate{
		TaskType:       "configUpdate",
		UpdateID:       up.UpdateID,
		ConfigFilePath: up.Config,
	}
	event := &fsm.Event{
		Type:   "ConfigUpdate",
		Action: api.ActionSuccess,
	}

	defer func() {
		// report upgrade result to cloudhub
		if err = util.ReportTaskResult(edgeConfigure, configUpdate.TaskType, configUpdate.UpdateID, *event); err != nil {
			klog.Errorf("failed to report config update result to cloud: %v", err)
		}
		// cleanup idempotency record
		if err = os.Remove(idempotencyRecord); err != nil {
			klog.Errorf("failed to remove idempotency_record file(%s): %v", idempotencyRecord, err)
		}
	}()

	// only allow update when last update finished
	if util.FileExists(idempotencyRecord) {
		event.Action = api.ActionFailure
		event.Msg = "last config update not finished, not allowed update again"
		return fmt.Errorf(event.Msg)
	}

	// create idempotency_record file
	if err := os.MkdirAll(filepath.Dir(idempotencyRecord), 0750); err != nil {
		reason := fmt.Sprintf("failed to create idempotency_record dir: %v", err)
		event.Action = api.ActionFailure
		event.Msg = reason
		return fmt.Errorf(reason)
	}
	if _, err := os.Create(idempotencyRecord); err != nil {
		reason := fmt.Sprintf("failed to create idempotency_record file: %v", err)
		event.Action = api.ActionFailure
		event.Msg = reason
		return fmt.Errorf(reason)
	}

	err = up.Backup()
	if err != nil {
		event.Action = api.ActionFailure
		event.Msg = err.Error()
		return err
	}

	err = common.Write2File(up.ConfigPath, edgeConfigure)
	if err != nil {
		return fmt.Errorf("failed to write new config : %v", err)
	}

	command := "sudo systemctl restart edgecore.service"
	cmd := util.NewCommand(command)
	err = cmd.Exec()
	if err != nil {
		event.Type = "Rollback"
		rbErr := up.Rollback()
		if rbErr != nil {
			event.Action = api.ActionFailure
			event.Msg = rbErr.Error()
		} else {
			event.Msg = err.Error()
		}
		return fmt.Errorf("config update process failed: %v", err)
	}

	return nil
}

func (up *ConfigUpdateOptions) Rollback() error {
	return rollback(up.NodeVersion, up.OldConfig.DataBase.DataSource, up.ConfigPath)
}

func (up *ConfigUpdateOptions) Backup() error {
	backupPath := filepath.Join(util.KubeEdgeBackupPath, up.NodeVersion)
	if err := os.MkdirAll(backupPath, 0750); err != nil {
		return fmt.Errorf("mkdirall failed: %v", err)
	}

	// backup edgecore.db: copy from origin path to backup path
	if err := copyFile(up.OldConfig.DataBase.DataSource, filepath.Join(backupPath, "edgecore.db")); err != nil {
		return fmt.Errorf("failed to backup db: %v", err)
	}
	// backup edgecore.yaml: copy from origin path to backup path
	if err := copyFile(up.ConfigPath, filepath.Join(backupPath, "edgecore.yaml")); err != nil {
		return fmt.Errorf("failed to back config: %v", err)
	}
	// backup edgecore: copy from origin path to backup path
	if err := copyFile(filepath.Join(util.KubeEdgeUsrBinPath, util.KubeEdgeBinaryName), filepath.Join(backupPath, util.KubeEdgeBinaryName)); err != nil {
		return fmt.Errorf("failed to backup edgecore: %v", err)
	}
	return nil
}

type ConfigUpdateOptions struct {
	UpdateID    string
	NodeVersion string
	ConfigPath  string
	OldConfig   *v1alpha2.EdgeCoreConfig
	Config      string
	Sets        string
	PreRun      string
	PostRun     string
}

type ConfigUpdate struct {
	UpdateID       string
	ConfigFilePath string
	TaskType       string

	Status string
	Reason string
}
