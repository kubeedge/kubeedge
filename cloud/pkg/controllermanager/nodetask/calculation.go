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

package nodetask

import (
	"strconv"

	"github.com/shopspring/decimal"
	"k8s.io/klog/v2"

	operationsv1alpha2 "github.com/kubeedge/api/apis/operations/v1alpha2"
)

// CalculatePhaseWithCounts calculates the node job phase based on the statistics of
// the node task phase.
func CalculatePhaseWithCounts(total, proc, fail int64,
	failureTolerateSpec string,
) operationsv1alpha2.JobPhase {
	if total == 0 {
		klog.Error("node task status total is 0")
		return operationsv1alpha2.JobPhaseFailure
	}
	// As long as there are nodes being processed, the task status must be in-progress.
	if proc > 0 {
		return operationsv1alpha2.JobPhaseInProgress
	}
	// If failureTolerate is not specified, all node tasks must succeed.
	var failureTolerate float64 = 0
	if failureTolerateSpec != "" {
		parsed, err := strconv.ParseFloat(failureTolerateSpec, 64)
		if err != nil {
			klog.Errorf("failed to parse failureTolerate, use default value 1, err: %v", err)
		} else {
			failureTolerate = parsed
		}
	}
	// fail / total > failureTolerate
	if fail > 0 && decimal.NewFromInt(fail).
		Div(decimal.NewFromInt(total)).
		Round(2).
		Cmp(decimal.NewFromFloat(failureTolerate)) == 1 {
		return operationsv1alpha2.JobPhaseFailure
	}
	// succ == total || fail / total <= failureTolerate
	return operationsv1alpha2.JobPhaseComplated
}
