/*
Copyright 2026 The KubeEdge Authors.

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
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	operationsv1alpha2 "github.com/kubeedge/api/apis/operations/v1alpha2"
)

const (
	nodeUpgradeJobReasonInitialized        = "Initialized"
	nodeUpgradeJobReasonInProgress         = "InProgress"
	nodeUpgradeJobReasonCompleted          = "Completed"
	nodeUpgradeJobReasonFailed             = "Failed"
	nodeUpgradeJobReasonPartiallySucceeded = "PartiallySucceeded"
	nodeUpgradeJobReasonTimedOut           = "TimedOut"
	nodeUpgradeJobReasonNodeVerifyFailed   = "NodeVerificationFailed"
)

func setNodeUpgradeJobCondition(
	job *operationsv1alpha2.NodeUpgradeJob,
	conditionType string,
	status metav1.ConditionStatus,
	reason, message string,
) bool {
	return meta.SetStatusCondition(&job.Status.Conditions, metav1.Condition{
		Type:               conditionType,
		Status:             status,
		Reason:             reason,
		Message:            message,
		ObservedGeneration: job.Generation,
	})
}

func setNodeUpgradeJobLifecycleConditions(
	job *operationsv1alpha2.NodeUpgradeJob,
	phase operationsv1alpha2.JobPhase,
	failedCount int64,
) bool {
	var changed bool
	changed = setNodeUpgradeJobCondition(job,
		operationsv1alpha2.NodeUpgradeJobConditionInProgress,
		conditionStatus(phase == operationsv1alpha2.JobPhaseInProgress),
		nodeUpgradeJobReasonInProgress,
		"Node upgrade job has node tasks still in progress.") || changed
	changed = setNodeUpgradeJobCondition(job,
		operationsv1alpha2.NodeUpgradeJobConditionCompleted,
		conditionStatus(phase == operationsv1alpha2.JobPhaseCompleted),
		nodeUpgradeJobReasonCompleted,
		"Node upgrade job has reached a terminal completed phase.") || changed
	changed = setNodeUpgradeJobCondition(job,
		operationsv1alpha2.NodeUpgradeJobConditionFailed,
		conditionStatus(phase == operationsv1alpha2.JobPhaseFailure),
		nodeUpgradeJobReasonFailed,
		job.Status.Reason) || changed
	changed = setNodeUpgradeJobCondition(job,
		operationsv1alpha2.NodeUpgradeJobConditionPartiallySucceeded,
		conditionStatus(phase == operationsv1alpha2.JobPhaseCompleted && failedCount > 0),
		nodeUpgradeJobReasonPartiallySucceeded,
		"Node upgrade job completed with failed nodes within the failure tolerance.") || changed
	return changed
}

func conditionStatus(ok bool) metav1.ConditionStatus {
	if ok {
		return metav1.ConditionTrue
	}
	return metav1.ConditionFalse
}
