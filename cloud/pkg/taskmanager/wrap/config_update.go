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

type ConfigUpdateJobTask struct {
	Obj *operationsv1alpha2.ConfigUpdateJobNodeTaskStatus
}

// Check that ConfigUpdateJobTask implements the NodeJobTask interface
var _ NodeJobTask = (*ConfigUpdateJobTask)(nil)

func (task *ConfigUpdateJobTask) NodeName() string {
	return task.Obj.NodeName
}

func (task *ConfigUpdateJobTask) CanExecute() bool {
	// TODO: Consider whether the node tasks in the "InProgress" status should be execute again?
	return task.Obj.Phase == operationsv1alpha2.NodeTaskPhasePending
}

func (task *ConfigUpdateJobTask) Phase() operationsv1alpha2.NodeTaskPhase {
	return task.Obj.Phase
}

func (task *ConfigUpdateJobTask) SetPhase(phase operationsv1alpha2.NodeTaskPhase, reason ...string) {
	task.Obj.Phase = phase
	if len(reason) > 0 {
		task.Obj.Reason = reason[0]
	}
}

func (task *ConfigUpdateJobTask) Action() (*actionflow.Action, error) {
	return actionflow.FlowConfigUpdateJob.First, nil
}

func (task *ConfigUpdateJobTask) GetObject() any {
	return task.Obj
}

type ConfigUpdateJob struct {
	Obj *operationsv1alpha2.ConfigUpdateJob
}

// Check that ConfigUpdateJob implements the NodeJob interface
var _ NodeJob = (*ConfigUpdateJob)(nil)

func NewConfigUpdateJob(obj *operationsv1alpha2.ConfigUpdateJob) *ConfigUpdateJob {
	return &ConfigUpdateJob{Obj: obj}
}

func (job ConfigUpdateJob) Name() string {
	return job.Obj.Name
}

func (job ConfigUpdateJob) ResourceType() string {
	return operationsv1alpha2.ResourceConfigUpdateJob
}

func (job ConfigUpdateJob) Concurrency() int {
	return int(job.Obj.Spec.Concurrency)
}

func (job ConfigUpdateJob) Spec() any {
	return job.Obj.Spec
}

func (job ConfigUpdateJob) Tasks() []NodeJobTask {
	res := make([]NodeJobTask, 0, len(job.Obj.Status.NodeStatus))
	for i := range job.Obj.Status.NodeStatus {
		pitem := &job.Obj.Status.NodeStatus[i]
		res = append(res, &ConfigUpdateJobTask{Obj: pitem})
	}
	return res
}

func (job ConfigUpdateJob) GetObject() any {
	return job.Obj
}
