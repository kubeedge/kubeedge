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
	"os"

	"github.com/spf13/cobra"
	"k8s.io/klog/v2"
	"sigs.k8s.io/yaml"

	"github.com/kubeedge/api/apis/componentconfig/edgecore/v1alpha2"
	"github.com/kubeedge/kubeedge/keadm/cmd/keadm/app/cmd/util"
	upgrdeedge "github.com/kubeedge/kubeedge/pkg/upgrade/edge"
)

func NewEdgeConfigUpdate() *cobra.Command {
	var opts ConfigUpdateOptions
	executor := newConfigUpdateExecutor()

	cmd := &cobra.Command{
		Use:   "config-update",
		Short: "Update EdgeCore Configuration.",
		Long:  "Update EdgeCore Configuration.",
		RunE: func(_cmd *cobra.Command, _args []string) error {
			var err error
			defer func() {
				// Report the result of the config-update process.
				reporter := upgrdeedge.NewJSONFileReporter(upgrdeedge.EventTypeConfigUpdate,
					"", executor.currentVersion)
				if reperr := reporter.Report(err); reperr != nil {
					klog.Errorf("failed to report config update result: %v", reperr)
				}
				if err != OccupiedError {
					executor.release()
				}
			}()
			err = executor.prerun(opts)
			if err != nil {
				return err
			}
			err = executor.configUpdate(opts)
			if err != nil {
				return err
			}
			executor.runPostRunHook(opts.PostRun)
			return nil
		},
	}
	AddConfigUpdateFlags(cmd, &opts)
	return cmd
}

type configUpdateExecutor struct {
	baseUpgradeExecutor
}

func newConfigUpdateExecutor() configUpdateExecutor {
	return configUpdateExecutor{baseUpgradeExecutor: baseUpgradeExecutor{}}
}

func (executor *configUpdateExecutor) prerun(opts ConfigUpdateOptions) error {
	if err := executor.baseUpgradeExecutor.prePreRun(opts.Config); err != nil {
		return err
	}
	if err := executor.baseUpgradeExecutor.postPreRun(opts.PreRun); err != nil {
		return err
	}
	return nil
}

func (executor *configUpdateExecutor) configUpdate(opts ConfigUpdateOptions) error {
	data, err := os.ReadFile(opts.Config)
	if err != nil {
		return fmt.Errorf("failed to read configfile %s, err: %v", opts.Config, err)
	}
	edgeConfigure := &v1alpha2.EdgeCoreConfig{}
	err = yaml.Unmarshal(data, edgeConfigure)
	if err != nil {
		return fmt.Errorf("failed to unmarshal configfile %s, err: %v", opts.Config, err)
	}

	err = util.ParseSet(edgeConfigure, opts.Sets)
	if err != nil {
		return fmt.Errorf("failed to parse sets value to config file: %v", err)
	}

	err = edgeConfigure.WriteTo(opts.Config)
	if err != nil {
		return fmt.Errorf("failed to write new edgecore config: %v", err)
	}

	cmd := util.NewCommand("sudo systemctl restart edgecore.service")
	err = cmd.Exec()
	if err != nil {
		return fmt.Errorf("failed restart edgecore %v", err)
	}

	return nil
}
