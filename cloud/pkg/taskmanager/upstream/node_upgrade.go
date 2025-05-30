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

package upstream

import (
	"context"
	"fmt"
	"sync"

	"github.com/go-logr/logr"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/klog/v2"

	operationsv1alpha2 "github.com/kubeedge/api/apis/operations/v1alpha2"
	crdcliset "github.com/kubeedge/api/client/clientset/versioned"
	"github.com/kubeedge/kubeedge/cloud/pkg/common/client"
	"github.com/kubeedge/kubeedge/cloud/pkg/taskmanager/status"
	"github.com/kubeedge/kubeedge/pkg/nodetask/actionflow"
	taskmsg "github.com/kubeedge/kubeedge/pkg/nodetask/message"
)

type NodeUpgradeJobHandler struct {
	logger logr.Logger
	crdcli crdcliset.Interface
}

// Check whether NodeUpgradeJobHandler implements UpstreamHandler interface.
var _ UpstreamHandler = (*NodeUpgradeJobHandler)(nil)

// newNodeUpgradeJobHandler creates a new NodeUpgradeJobHandler.
func newNodeUpgradeJobHandler(ctx context.Context) *NodeUpgradeJobHandler {
	logger := klog.FromContext(ctx).
		WithName(fmt.Sprintf("upstream-%s", operationsv1alpha2.ResourceNodeUpgradeJob))
	return &NodeUpgradeJobHandler{
		logger: logger,
		crdcli: client.GetCRDClient(),
	}
}

func (h NodeUpgradeJobHandler) Logger() logr.Logger {
	return h.logger
}

func (NodeUpgradeJobHandler) GetAction(name string) *actionflow.Action {
	return actionflow.FlowNodeUpgradeJob.Find(name)
}

func (h *NodeUpgradeJobHandler) UpdateNodeTaskStatus(
	jobName, nodeName string,
	isFinalAction bool,
	upmsg taskmsg.UpstreamMessage,
) error {
	var (
		actoinStatus operationsv1alpha2.NodeUpgradeJobActionStatus
		err          error
		wg           sync.WaitGroup
	)

	actoinStatus.Action = operationsv1alpha2.NodeUpgradeJobAction(upmsg.Action)
	if upmsg.Succ {
		actoinStatus.Status = metav1.ConditionTrue
	} else {
		actoinStatus.Status = metav1.ConditionFalse
		actoinStatus.Reason = upmsg.Reason
	}
	actoinStatus.Time = upmsg.FinishTime

	phase := operationsv1alpha2.NodeTaskPhaseInProgress
	if isFinalAction {
		if upmsg.Succ {
			phase = operationsv1alpha2.NodeTaskPhaseSuccessful
		} else {
			phase = operationsv1alpha2.NodeTaskPhaseFailure
		}
	}

	wg.Add(1)
	opts := status.UpdateStatusOptions{
		TryUpdateStatusOptions: status.TryUpdateStatusOptions{
			JobName:      jobName,
			NodeName:     nodeName,
			Phase:        phase,
			ExtendInfo:   upmsg.Extend,
			ActionStatus: &actoinStatus,
		},
		Callback: func(err error) {
			if err != nil {
				err = fmt.Errorf("failed to update image prepull job status, err: %v", err)
			}
			wg.Done()
		},
	}
	status.GetNodeUpgradeJobStatusUpdater().UpdateStatus(opts)
	wg.Wait()
	return err
}

// func (h *NodeUpgradeJobHandler) FindNodeTaskStatus(ctx context.Context, res taskmsg.Resource,
// ) (int, any, error) {
// 	job, err := h.crdcli.OperationsV1alpha2().NodeUpgradeJobs().
// 		Get(ctx, res.JobName, metav1.GetOptions{})
// 	if err != nil {
// 		return -1, nil, fmt.Errorf("failed to get node upgrade job, err: %v", err)
// 	}
// 	idx := -1
// 	for i, st := range job.Status.NodeStatus {
// 		if st.NodeName == res.NodeName {
// 			idx = i
// 			break
// 		}
// 	}
// 	var nodetask *operationsv1alpha2.NodeUpgradeJobNodeTaskStatus
// 	if idx >= 0 {
// 		nodetask = &job.Status.NodeStatus[idx]
// 	}
// 	return idx, nodetask, nil
// }

// func (NodeUpgradeJobHandler) ConvToNodeTask(nodename string, upmsg *taskmsg.UpstreamMessage,
// ) (wrap.NodeJobTask, error) {
// 	obj := &operationsv1alpha2.NodeUpgradeJobNodeTaskStatus{
// 		BasicNodeTaskStatus: operationsv1alpha2.BasicNodeTaskStatus{
// 			NodeName: nodename,
// 		},
// 		Action: operationsv1alpha2.NodeUpgradeJobAction(upmsg.Action),
// 	}
// 	if !upmsg.Succ {
// 		obj.Phase = operationsv1alpha2.NodeTaskPhaseFailure
// 		obj.Reason = upmsg.Reason
// 	} else {
// 		obj.Phase = operationsv1alpha2.NodeTaskPhaseInProgress
// 	}
// 	if upmsg.Extend != "" {
// 		fromVer, toVer, err := taskmsg.ParseNodeUpgradeJobExtend(upmsg.Extend)
// 		if err != nil {
// 			return nil, fmt.Errorf("failed to parse node upgrade job extend, err: %v", err)
// 		}
// 		obj.HistoricVersion = fromVer
// 		obj.CurrentVersion = toVer
// 	}
// 	return &wrap.NodeUpgradeJobTask{Obj: obj}, nil
// }

// func (h *NodeUpgradeJobHandler) GetCurrentAction(nodetask any) (*actionflow.Action, error) {
// 	obj, ok := nodetask.(*operationsv1alpha2.NodeUpgradeJobNodeTaskStatus)
// 	if !ok {
// 		return nil, fmt.Errorf("failed to convert nodetask to NodeUpgradeJobNodeTaskStatus, "+
// 			"invalid type: %T", nodetask)
// 	}
// 	action := actionflow.FlowNodeUpgradeJob.Find(string(obj.Action))
// 	if action == nil {
// 		return nil, fmt.Errorf("invalid action %s", obj.Action)
// 	}
// 	return action, nil
// }

// func (h *NodeUpgradeJobHandler) UpdateNodeTaskStatus(jobname string, nodetask wrap.NodeJobTask) error {
// 	obj, ok := nodetask.GetObject().(*operationsv1alpha2.NodeUpgradeJobNodeTaskStatus)
// 	if !ok {
// 		return fmt.Errorf("failed to convert nodetask to NodeUpgradeJobNodeTaskStatus, "+
// 			"invalid type: %T", nodetask)
// 	}
// 	var (
// 		err error
// 		wg  sync.WaitGroup
// 	)
// 	wg.Add(1)
// 	opts := status.UpdateStatusOptions[operationsv1alpha2.NodeUpgradeJobNodeTaskStatus]{
// 		JobName:        jobname,
// 		NodeTaskStatus: *obj,
// 		Callback: func(err error) {
// 			if err != nil {
// 				err = fmt.Errorf("failed to update node upgrade job status, err: %v", err)
// 			}
// 			wg.Done()
// 		},
// 	}
// 	status.GetNodeUpgradeJobStatusUpdater().UpdateStatus(opts)
// 	wg.Wait()
// 	return err
// }
