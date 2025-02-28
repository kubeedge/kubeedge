package wrap

import (
	"fmt"

	operationsv1alpha2 "github.com/kubeedge/api/apis/operations/v1alpha2"
	"github.com/kubeedge/kubeedge/pkg/nodetask/actionflow"
)

type NodeUpgradeJobTask struct {
	Obj *operationsv1alpha2.NodeUpgradeJobNodeTaskStatus
}

// Check that NodeUpgradeJobTask implements the NodeJobTask interface
var _ NodeJobTask = (*NodeUpgradeJobTask)(nil)

func (task NodeUpgradeJobTask) NodeName() string {
	return task.Obj.NodeName
}

func (task NodeUpgradeJobTask) CanExecute() bool {
	// TODO: Consider whether the node tasks in the "InProgress" status should be execute again?
	return task.Obj.Status == operationsv1alpha2.NodeTaskStatusPending
}

func (task NodeUpgradeJobTask) Status() operationsv1alpha2.NodeTaskStatus {
	return task.Obj.Status
}

func (task *NodeUpgradeJobTask) SetStatus(status operationsv1alpha2.NodeTaskStatus) {
	task.Obj.Status = status
}

func (task NodeUpgradeJobTask) Action() (*actionflow.Action, error) {
	action := actionflow.FlowNodeUpgradeJob.Find(string(task.Obj.Action))
	if action == nil {
		return nil, fmt.Errorf("no valid node upgrade job action '%s' was found", task.Obj.Action)
	}
	return action, nil
}

func (task *NodeUpgradeJobTask) SetAction(action *actionflow.Action) {
	task.Obj.Action = operationsv1alpha2.NodeUpgradeJobAction(action.Name)
}

func (task NodeUpgradeJobTask) GetObject() any {
	return task.Obj
}

type NodeUpgradeJob struct {
	obj *operationsv1alpha2.NodeUpgradeJob
}

// Check that NodeUpgradeJob implements the NodeJob interface
var _ NodeJob = (*NodeUpgradeJob)(nil)

func NewNodeUpgradeJob(obj *operationsv1alpha2.NodeUpgradeJob) *NodeUpgradeJob {
	return &NodeUpgradeJob{obj: obj}
}

func (job NodeUpgradeJob) Name() string {
	return job.obj.Name
}

func (job NodeUpgradeJob) ResourceType() string {
	return operationsv1alpha2.ResourceNodeUpgradeJob
}

func (job NodeUpgradeJob) Concurrency() int {
	return int(job.obj.Spec.Concurrency)
}

func (job NodeUpgradeJob) Spec() any {
	return job.obj.Spec
}

func (job NodeUpgradeJob) Tasks() []NodeJobTask {
	res := make([]NodeJobTask, 0, len(job.obj.Status.NodeStatus))
	for _, it := range job.obj.Status.NodeStatus {
		res = append(res, &NodeUpgradeJobTask{Obj: &it})
	}
	return res
}

func (job NodeUpgradeJob) GetObject() any {
	return job.obj
}
