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
	"encoding/json"
	"errors"
	"fmt"
	"sync"

	"github.com/go-logr/logr"
	"k8s.io/klog/v2"

	operationsv1alpha2 "github.com/kubeedge/api/apis/operations/v1alpha2"
	crdcliset "github.com/kubeedge/api/client/clientset/versioned"
	"github.com/kubeedge/kubeedge/cloud/pkg/common/client"
	"github.com/kubeedge/kubeedge/cloud/pkg/taskmanager/v1alpha2/executor"
	"github.com/kubeedge/kubeedge/cloud/pkg/taskmanager/v1alpha2/status"
	"github.com/kubeedge/kubeedge/cloud/pkg/taskmanager/v1alpha2/wrap"
	"github.com/kubeedge/kubeedge/pkg/nodetask/actionflow"
	taskmsg "github.com/kubeedge/kubeedge/pkg/nodetask/message"
)

type ImagePrePullJobHandler struct {
	logger logr.Logger
	crdcli crdcliset.Interface
}

// Check that ImagePrePullJobHandler implements UpstreamHandler interface.
var _ UpstreamHandler = (*ImagePrePullJobHandler)(nil)

// newImagePrePullJobHandler creates a new ImagePrePullJobHandler.
func newImagePrePullJobHandler(ctx context.Context) *ImagePrePullJobHandler {
	logger := klog.FromContext(ctx).
		WithName(fmt.Sprintf("upstream-%s", operationsv1alpha2.ResourceImagePrePullJob))
	return &ImagePrePullJobHandler{
		logger: logger,
		crdcli: client.GetCRDClient(),
	}
}

func (h *ImagePrePullJobHandler) Logger() logr.Logger {
	return h.logger
}

func (h *ImagePrePullJobHandler) ConvToNodeTask(nodename string, upmsg *taskmsg.UpstreamMessage,
) (wrap.NodeJobTask, error) {
	obj := &operationsv1alpha2.ImagePrePullNodeTaskStatus{
		BasicNodeTaskStatus: operationsv1alpha2.BasicNodeTaskStatus{
			NodeName: nodename,
		},
		Action: operationsv1alpha2.ImagePrePullJobAction(upmsg.Action),
	}
	if !upmsg.Succ {
		obj.Phase = operationsv1alpha2.NodeTaskPhaseFailure
		obj.Reason = upmsg.Reason
	} else {
		obj.Phase = operationsv1alpha2.NodeTaskPhaseInProgress
	}
	h.logger.V(3).Info("convert extend message", "action", obj.Action, "extend", upmsg.Extend)
	if obj.Action == operationsv1alpha2.ImagePrePullJobActionPull && upmsg.Extend != "" {
		imageStatus := make([]operationsv1alpha2.ImageStatus, 0)
		if err := json.Unmarshal([]byte(upmsg.Extend), &imageStatus); err != nil {
			return nil, fmt.Errorf("failed to unmarshal image mapper, err: %v", err)
		}
		obj.ImageStatus = imageStatus
	}
	return &wrap.ImagePrePullJobTask{Obj: obj}, nil
}

func (h *ImagePrePullJobHandler) GetCurrentAction(nodetask any) (*actionflow.Action, error) {
	obj, ok := nodetask.(*operationsv1alpha2.ImagePrePullNodeTaskStatus)
	if !ok {
		return nil, fmt.Errorf("failed to convert nodetask to ImagePrePullNodeTaskStatus, "+
			"invalid type: %T", nodetask)
	}
	action := actionflow.FlowImagePrePullJob.Find(string(obj.Action))
	if action == nil {
		return nil, fmt.Errorf("invalid action %s", obj.Action)
	}
	return action, nil
}

func (h *ImagePrePullJobHandler) ReleaseExecutorConcurrent(res taskmsg.Resource) error {
	exec, err := executor.GetExecutor(res.ResourceType, res.JobName)
	if err != nil && !errors.Is(err, executor.ErrExecutorNotExists) {
		return fmt.Errorf("failed to get executor, err: %v", err)
	}
	if exec != nil {
		exec.FinishTask()
	}
	return nil
}

func (h *ImagePrePullJobHandler) UpdateNodeTaskStatus(jobname string, nodetask wrap.NodeJobTask) error {
	obj, ok := nodetask.GetObject().(*operationsv1alpha2.ImagePrePullNodeTaskStatus)
	if !ok {
		return fmt.Errorf("failed to convert nodetask to ImagePrePullNodeTaskStatus, "+
			"invalid type: %T", nodetask)
	}
	var (
		err error
		wg  sync.WaitGroup
	)
	wg.Add(1)
	opts := status.UpdateStatusOptions[operationsv1alpha2.ImagePrePullNodeTaskStatus]{
		JobName:        jobname,
		NodeTaskStatus: *obj,
		Callback: func(err error) {
			if err != nil {
				err = fmt.Errorf("failed to update image prepull job status, err: %v", err)
			}
			wg.Done()
		},
	}
	status.GetImagePrePullJobStatusUpdater().UpdateStatus(opts)
	wg.Wait()
	return err
}
