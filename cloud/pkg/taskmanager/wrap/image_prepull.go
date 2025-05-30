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

type ImagePrePullJobTask struct {
	Obj *operationsv1alpha2.ImagePrePullNodeTaskStatus
}

// Check whether ImagePrePullJobTask implements the NodeJobTask interface
var _ NodeJobTask = (*ImagePrePullJobTask)(nil)

func (task ImagePrePullJobTask) NodeName() string {
	return task.Obj.NodeName
}

func (task ImagePrePullJobTask) CanExecute() bool {
	// TODO: Consider whether the node tasks in the "InProgress" status should be execute again?
	return task.Obj.Phase == operationsv1alpha2.NodeTaskPhasePending
}

func (task ImagePrePullJobTask) Phase() operationsv1alpha2.NodeTaskPhase {
	return task.Obj.Phase
}

func (task *ImagePrePullJobTask) SetPhase(phase operationsv1alpha2.NodeTaskPhase, reason ...string) {
	task.Obj.Phase = phase
	if len(reason) > 0 {
		task.Obj.Reason = reason[0]
	}
}

func (task ImagePrePullJobTask) Action() (*actionflow.Action, error) {
	return actionflow.FlowImagePrePullJob.First, nil
}

func (task ImagePrePullJobTask) GetObject() any {
	return task.Obj
}

type ImagePrePullJob struct {
	Obj *operationsv1alpha2.ImagePrePullJob
}

// Check whether ImagePrePullJob implements the NodeJob interface
var _ NodeJob = (*ImagePrePullJob)(nil)

func NewImagePrepullJob(obj *operationsv1alpha2.ImagePrePullJob) *ImagePrePullJob {
	return &ImagePrePullJob{Obj: obj}
}

func (job ImagePrePullJob) Name() string {
	return job.Obj.Name
}

func (job ImagePrePullJob) ResourceType() string {
	return operationsv1alpha2.ResourceImagePrePullJob
}

func (job ImagePrePullJob) Concurrency() int {
	return int(job.Obj.Spec.ImagePrePullTemplate.Concurrency)
}

func (job ImagePrePullJob) Spec() any {
	return job.Obj.Spec
}

func (job ImagePrePullJob) Tasks() []NodeJobTask {
	res := make([]NodeJobTask, 0, len(job.Obj.Status.NodeStatus))
	for i := range job.Obj.Status.NodeStatus {
		pitem := &job.Obj.Status.NodeStatus[i]
		res = append(res, &ImagePrePullJobTask{Obj: pitem})
	}
	return res
}

func (job ImagePrePullJob) GetObject() any {
	return job.Obj
}
