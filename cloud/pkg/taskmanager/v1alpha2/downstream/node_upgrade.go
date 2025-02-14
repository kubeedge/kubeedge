package downstream

import (
	"context"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	operationsv1alpha2 "github.com/kubeedge/api/apis/operations/v1alpha2"
	"github.com/kubeedge/kubeedge/cloud/pkg/common/messagelayer"
	"github.com/kubeedge/kubeedge/pkg/nodetask/actionflow"
	"github.com/kubeedge/kubeedge/pkg/nodetask/message"
)

type NodeUpgradeJobTask struct {
	obj *operationsv1alpha2.NodeUpgradeJobNodeTaskStatus
}

var _ NodeTask = (*NodeUpgradeJobTask)(nil)

func (task NodeUpgradeJobTask) NodeName() string {
	return task.obj.NodeName
}

func (task NodeUpgradeJobTask) CanExecute() bool {
	if task.obj.Action == operationsv1alpha2.NodeUpgradeJobActionInit &&
		task.obj.Status == metav1.ConditionTrue {
		return true
	}
	// For retry situation. The restart of CloudCore will lose the progress in memory,
	// so node tasks that have not obtained the action results need to be retried, and
	// the idempotent processing is handled by EdgeCore.
	return task.obj.Status == ""
}

type UpdateJob func(ctx context.Context, obj **operationsv1alpha2.NodeUpgradeJob) error

type NodeUpgradeJob struct {
	obj          *operationsv1alpha2.NodeUpgradeJob
	messageLayer messagelayer.MessageLayer
	updateJobFun UpdateJob
}

var _ NodeTaskDownstream = (*NodeUpgradeJob)(nil)

func NewNodeUpgradeJob(
	obj *operationsv1alpha2.NodeUpgradeJob,
	updateJobFun UpdateJob,
) *NodeUpgradeJob {
	return &NodeUpgradeJob{
		obj:          obj,
		messageLayer: messagelayer.TaskManagerMessageLayer(),
		updateJobFun: updateJobFun,
	}
}

func (down NodeUpgradeJob) JobName() string {
	return down.obj.Name
}

func (down *NodeUpgradeJob) Tasks(_ctx context.Context) []NodeTask {
	res := make([]NodeTask, 0, len(down.obj.Status.NodeStatus))
	for _, it := range down.obj.Status.NodeStatus {
		res = append(res, &NodeUpgradeJobTask{obj: &it})
	}
	return res
}

func (down *NodeUpgradeJob) SendTaskToEdge(ctx context.Context, task NodeTask) error {
	taskimpl, ok := task.(*NodeUpgradeJobTask)
	if !ok {
		// TODO: handle error
	}
	op := string(taskimpl.obj.Action)
	if taskimpl.obj.Action == operationsv1alpha2.NodeUpgradeJobActionInit {
		action := actionflow.FlowNodeUpgradeJob.Find(string(taskimpl.obj.Action))
		if action == nil || action.Next(true) == nil {
			// TODO: handle error
		}
		op = action.Next(true).Name
	}
	msgres := message.Resource{
		APIVersion:   operationsv1alpha2.SchemeGroupVersion.String(),
		ResourceType: operationsv1alpha2.ResourceNodeUpgradeJob,
		TaskName:     down.obj.Name,
		Node:         taskimpl.obj.NodeName,
	}
	msg := messagelayer.BuildNodeTaskRouter(msgres, op).FillBody(down.obj.Spec)
	if err := down.messageLayer.Send(*msg); err != nil {
		// TODO: handle error
	}
	return nil
}

func (down *NodeUpgradeJob) HandleNodeTaskError(ctx context.Context, task NodeTask, err error) {
	// TODO: ...
}
