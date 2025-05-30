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

type ImagePrePullJobHandler struct {
	logger logr.Logger
	crdcli crdcliset.Interface
}

// Check whether ImagePrePullJobHandler implements UpstreamHandler interface.
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

func (h ImagePrePullJobHandler) Logger() logr.Logger {
	return h.logger
}

func (ImagePrePullJobHandler) GetAction(name string) *actionflow.Action {
	return actionflow.FlowImagePrePullJob.Find(name)
}

func (h *ImagePrePullJobHandler) UpdateNodeTaskStatus(
	jobName, nodeName string,
	isFinalAction bool,
	upmsg taskmsg.UpstreamMessage,
) error {
	var (
		actoinStatus operationsv1alpha2.ImagePrePullJobActionStatus
		err          error
		wg           sync.WaitGroup
	)

	actoinStatus.Action = operationsv1alpha2.ImagePrePullJobAction(upmsg.Action)
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
	status.GetImagePrePullJobStatusUpdater().UpdateStatus(opts)
	wg.Wait()
	return err
}
