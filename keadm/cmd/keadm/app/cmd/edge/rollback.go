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
	"os"

	"github.com/spf13/cobra"
	"k8s.io/klog/v2"
	"sigs.k8s.io/yaml"

	"github.com/kubeedge/api/apis/common/constants"
	"github.com/kubeedge/api/apis/componentconfig/edgecore/v1alpha2"
	api "github.com/kubeedge/api/apis/fsm/v1alpha1"
	"github.com/kubeedge/kubeedge/keadm/cmd/keadm/app/cmd/util"
	"github.com/kubeedge/kubeedge/pkg/util/fsm"
	"github.com/kubeedge/kubeedge/pkg/version"
)

// NewEdgeUpgrade returns KubeEdge edge upgrade command.
func NewEdgeRollback() *cobra.Command {
	rollbackOptions := newRollbackOptions()

	cmd := &cobra.Command{
		Use:   "rollback",
		Short: "rollback edge component. Rollback the edge node to the desired version.",
		Long:  "Rollback edge component. Rollback the edge node to the desired version.",
		RunE: func(_ *cobra.Command, _ []string) error {
			// rollback edge core
			return rollbackEdgeCore(rollbackOptions)
		},
	}

	addRollbackFlags(cmd, rollbackOptions)
	return cmd
}

// newJoinOptions returns a struct ready for being used for creating cmd join flags.
func newRollbackOptions() *RollbackOptions {
	opts := &RollbackOptions{}
	opts.HistoryVersion = version.Get().String()
	opts.Config = constants.DefaultConfigDir + "edgecore.yaml"

	return opts
}

func rollbackEdgeCore(ro *RollbackOptions) error {
	// get EdgeCore configuration from edgecore.yaml config file
	data, err := os.ReadFile(ro.Config)
	if err != nil {
		return fmt.Errorf("failed to read config file %s: %v", ro.Config, err)
	}

	configure := &v1alpha2.EdgeCoreConfig{}
	err = yaml.Unmarshal(data, configure)
	if err != nil {
		return fmt.Errorf("failed to unmarshal config file %s: %v", ro.Config, err)
	}
	event := &fsm.Event{
		Type:   "RollBack",
		Action: api.ActionSuccess,
	}
	defer func() {
		// report upgrade result to cloudhub
		if err = util.ReportTaskResult(configure, ro.TaskType, ro.TaskName, *event); err != nil {
			klog.Warningf("failed to report upgrade result to cloud: %v", err)
		}
	}()

	rbErr := rollback(ro.HistoryVersion, configure.DataBase.DataSource, ro.Config)
	if rbErr != nil {
		event.Action = api.ActionFailure
		event.Msg = fmt.Sprintf("upgrade error: %v, rollback error: %v", err, rbErr)
	}

	return nil
}

type RollbackOptions struct {
	HistoryVersion string
	TaskType       string
	TaskName       string
	Config         string
}

func addRollbackFlags(cmd *cobra.Command, rollbackOptions *RollbackOptions) {
	cmd.Flags().StringVar(&rollbackOptions.HistoryVersion, "history", rollbackOptions.HistoryVersion,
		"Use this key to specify the origin version before upgrade")

	cmd.Flags().StringVar(&rollbackOptions.Config, "config", rollbackOptions.Config,
		"Use this key to specify the path to the edgecore configuration file.")

	cmd.Flags().StringVar(&rollbackOptions.TaskType, "type", "rollback",
		"Use this key to specify the task type for reporting status.")

	cmd.Flags().StringVar(&rollbackOptions.TaskName, "name", rollbackOptions.TaskName,
		"Use this key to specify the task name for reporting status.")
}
