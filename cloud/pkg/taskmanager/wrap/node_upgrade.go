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

package wrap

import (
	operationsv1alpha2 "github.com/kubeedge/api/apis/operations/v1alpha2"
	"github.com/kubeedge/kubeedge/pkg/nodetask/actionflow"
)

type NodeUpgradeJobTask struct {
	Obj *operationsv1alpha2.NodeUpgradeJobNodeTaskStatus
}

// Check whether NodeUpgradeJobTask implements the NodeJobTask interface
var _ NodeJobTask = (*NodeUpgradeJobTask)(nil)

func (task NodeUpgradeJobTask) NodeName() string {
	return task.Obj.NodeName
}

func (task NodeUpgradeJobTask) CanExecute() bool {
	// TODO: Consider whether the node tasks in the "InProgress" status should be execute again?
	return task.Obj.Phase == operationsv1alpha2.NodeTaskPhasePending
}

func (task NodeUpgradeJobTask) Phase() operationsv1alpha2.NodeTaskPhase {
	return task.Obj.Phase
}

func (task *NodeUpgradeJobTask) SetPhase(phase operationsv1alpha2.NodeTaskPhase, reason ...string) {
	task.Obj.Phase = phase
	if len(reason) > 0 {
		task.Obj.Reason = reason[0]
	}
}

func (task NodeUpgradeJobTask) Action() (*actionflow.Action, error) {
	return actionflow.FlowNodeUpgradeJob.First, nil
}

func (task NodeUpgradeJobTask) GetObject() any {
	return task.Obj
}

type NodeUpgradeJob struct {
	Obj *operationsv1alpha2.NodeUpgradeJob
}

// Check whether NodeUpgradeJob implements the NodeJob interface
var _ NodeJob = (*NodeUpgradeJob)(nil)

func NewNodeUpgradeJob(obj *operationsv1alpha2.NodeUpgradeJob) *NodeUpgradeJob {
	return &NodeUpgradeJob{Obj: obj}
}

func (job NodeUpgradeJob) Name() string {
	return job.Obj.Name
}

func (job NodeUpgradeJob) ResourceType() string {
	return operationsv1alpha2.ResourceNodeUpgradeJob
}

func (job NodeUpgradeJob) Concurrency() int {
	return int(job.Obj.Spec.Concurrency)
}

func (job NodeUpgradeJob) Spec() any {
	return job.Obj.Spec
}

func (job NodeUpgradeJob) Tasks() []NodeJobTask {
	res := make([]NodeJobTask, 0, len(job.Obj.Status.NodeStatus))
	for i := range job.Obj.Status.NodeStatus {
		pitem := &job.Obj.Status.NodeStatus[i]
		res = append(res, &NodeUpgradeJobTask{Obj: pitem})
	}
	return res
}

func (job NodeUpgradeJob) GetObject() any {
	return job.Obj
}
