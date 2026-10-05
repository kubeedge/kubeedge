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
	"strings"

	"github.com/spf13/cobra"
	"k8s.io/klog/v2"
	"sigs.k8s.io/yaml"

	"github.com/kubeedge/api/apis/componentconfig/edgecore/v1alpha2"
	"github.com/kubeedge/api/apis/componentconfig/edgecore/v1alpha2/validation"
	"github.com/kubeedge/kubeedge/keadm/cmd/keadm/app/cmd/helm"
	upgrdeedge "github.com/kubeedge/kubeedge/pkg/upgrade/edge"
	pkgutil "github.com/kubeedge/kubeedge/pkg/util"
	"github.com/kubeedge/kubeedge/pkg/util/execs"
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
	sets := strings.Split(opts.Sets, ",")
	mergedData, err := helm.MergeSetsToBytes(data, sets)
	if err != nil {
		return fmt.Errorf("failed to merge sets to edgecore's config with err:%v", err)
	}
	edgeConfigure := &v1alpha2.EdgeCoreConfig{}
	//Check if there are any unknown fields in the set.
	if err = yaml.UnmarshalStrict(mergedData, edgeConfigure); err != nil {
		return err
	}
	if errs := validation.ValidateEdgeCoreConfiguration(edgeConfigure); len(errs) > 0 {
		return errors.New(pkgutil.SpliceErrors(errs.ToAggregate().Errors()))
	}
	if err = writeFile(opts.Config, mergedData); err != nil {
		return err
	}

	cmd := execs.NewCommand("sudo systemctl restart edgecore.service")
	err = cmd.Exec()
	if err != nil {
		return fmt.Errorf("failed restart edgecore %v", err)
	}
	return nil
}

func writeFile(filename string, data []byte) error {
	fileInfo, err := os.Stat(filename)
	if err != nil {
		// If the file does not exist, the default permissions are 0644.
		return os.WriteFile(filename, data, 0644)
	}
	// Write the file using the original file permissions.
	return os.WriteFile(filename, data, fileInfo.Mode())
}
