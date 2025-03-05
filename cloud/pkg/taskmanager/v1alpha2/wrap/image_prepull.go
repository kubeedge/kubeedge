package wrap

import (
	"fmt"
	"time"

	operationsv1alpha2 "github.com/kubeedge/api/apis/operations/v1alpha2"
	"github.com/kubeedge/kubeedge/pkg/nodetask/actionflow"
)

type ImagePrePullJobTask struct {
	Obj *operationsv1alpha2.ImagePrePullNodeTaskStatus
}

// Check that ImagePrePullJobTask implements the NodeJobTask interface
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

func (task *ImagePrePullJobTask) ToSuccessful() {
	task.Obj.Phase = operationsv1alpha2.NodeTaskPhaseSuccessful
}

func (task *ImagePrePullJobTask) ToInProgress(t time.Time) {
	task.Obj.Phase = operationsv1alpha2.NodeTaskPhaseInProgress
	task.Obj.Time = t.UTC().Format(time.RFC3339)
}

func (task *ImagePrePullJobTask) ToFailure(reason string) {
	task.Obj.Phase = operationsv1alpha2.NodeTaskPhaseFailure
	task.Obj.Reason = reason
}

func (task ImagePrePullJobTask) Action() (*actionflow.Action, error) {
	action := actionflow.FlowImagePrePullJob.Find(string(task.Obj.Action))
	if action == nil {
		return nil, fmt.Errorf("no valid image prepull job action '%s' was found", task.Obj.Action)
	}
	return action, nil
}

func (task *ImagePrePullJobTask) SetAction(action *actionflow.Action) {
	task.Obj.Action = operationsv1alpha2.ImagePrePullJobAction(action.Name)
}

func (task ImagePrePullJobTask) GetObject() any {
	return task.Obj
}

type ImagePrePullJob struct {
	Obj *operationsv1alpha2.ImagePrePullJob
}

// Check that ImagePrePullJob implements the NodeJob interface
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
	for _, it := range job.Obj.Status.NodeStatus {
		res = append(res, &ImagePrePullJobTask{Obj: &it})
	}
	return res
}

func (job ImagePrePullJob) GetObject() any {
	return job.Obj
}
