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
	"context"
	"errors"
	"fmt"

	"k8s.io/klog/v2"

	cfgv1alpha2 "github.com/kubeedge/api/apis/componentconfig/edgecore/v1alpha2"
	"github.com/kubeedge/kubeedge/keadm/cmd/keadm/app/cmd/util"
	edgecoreutil "github.com/kubeedge/kubeedge/keadm/cmd/keadm/app/cmd/util/edgecore"
	"github.com/kubeedge/kubeedge/keadm/cmd/keadm/app/cmd/util/idempotency"
)

const unknownEdgeCoreVersion = "unknown"

var OccupiedError = errors.New("the mutually exclusive command is being executed")

type baseUpgradeExecutor struct {
	currentVersion string

	cfg *cfgv1alpha2.EdgeCoreConfig
}

func (executor *baseUpgradeExecutor) prePreRun(configpath string) error {
	ctx := context.Background()
	// Set the default value of currentVersion to unknown.
	executor.currentVersion = unknownEdgeCoreVersion

	// Determine whether there are mutually exclusive commands running. If not,
	// it occupies the resource and prevents other mutually exclusive commands from
	// running at the same time. Otherwise, return an error.
	occupied, err := idempotency.Occupy()
	if err != nil {
		return fmt.Errorf("failed to occupy command execution, err: %v", err)
	}
	if occupied {
		return OccupiedError
	}

	// Parse the edgecore config file.
	cfg, err := util.ParseEdgecoreConfig(configpath)
	if err != nil {
		return fmt.Errorf("failed to parse the edgecore config file %s, err: %v",
			configpath, err)
	}
	executor.cfg = cfg

	// Get the current version of edgecore.
	if ver := edgecoreutil.GetVersion(ctx, cfg); ver != "" {
		executor.currentVersion = ver
	}
	return nil
}

func (executor *baseUpgradeExecutor) postPreRun(prerunHook string) error {
	// If pre-run script is specified, execute it.
	if prerunHook != "" {
		fmt.Printf("Executing pre-run script: %s\n", prerunHook)
		if err := util.RunScript(prerunHook); err != nil {
			return fmt.Errorf("failed to run the pre-run script %s, err: %v", prerunHook, err)
		}
	}
	return nil
}

func (executor *baseUpgradeExecutor) release() {
	// Release the occupied resource.
	if idempotency.IsOccupied() {
		if err := idempotency.Release(); err != nil {
			klog.Errorf("failed to release the occupied command execution, err: %v", err)
		}
	}
}

func (executor *baseUpgradeExecutor) runPostRunHook(postrunHook string) {
	// If post-run script is specified, execute it.
	if postrunHook != "" {
		fmt.Printf("Executing post-run script: %s\n", postrunHook)
		if err := util.RunScript(postrunHook); err != nil {
			fmt.Printf("Execute post-run script: %s failed: %v\n", postrunHook, err)
		}
	}
}
