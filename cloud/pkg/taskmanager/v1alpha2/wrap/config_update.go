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
	"fmt"
	"time"

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

func (task *ConfigUpdateJobTask) ToSuccessful() {
	task.Obj.Phase = operationsv1alpha2.NodeTaskPhaseSuccessful
}

func (task *ConfigUpdateJobTask) ToInProgress(t time.Time) {
	task.Obj.Phase = operationsv1alpha2.NodeTaskPhaseInProgress
	task.Obj.Time = t.UTC().Format(time.RFC3339)
}

func (task *ConfigUpdateJobTask) ToFailure(reason string) {
	task.Obj.Phase = operationsv1alpha2.NodeTaskPhaseFailure
	task.Obj.Reason = reason
}

func (task *ConfigUpdateJobTask) Action() (*actionflow.Action, error) {
	if task.Obj.Action == "" {
		return actionflow.FlowConfigUpdateJob.First, nil
	}
	action := actionflow.FlowConfigUpdateJob.Find(string(task.Obj.Action))
	if action == nil {
		return nil, fmt.Errorf("no valid config update job action '%s' was found", task.Obj.Action)
	}
	return action, nil
}

func (task *ConfigUpdateJobTask) SetAction(action *actionflow.Action) {
	task.Obj.Action = operationsv1alpha2.ConfigUpdateJobAction(action.Name)
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
