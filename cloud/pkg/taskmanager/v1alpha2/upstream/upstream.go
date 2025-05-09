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
	"fmt"

	"github.com/go-logr/logr"
	"k8s.io/klog/v2"

	operationsv1alpha2 "github.com/kubeedge/api/apis/operations/v1alpha2"
	"github.com/kubeedge/beehive/pkg/core/model"
	"github.com/kubeedge/kubeedge/cloud/pkg/taskmanager/v1alpha2/wrap"
	taskmsg "github.com/kubeedge/kubeedge/pkg/nodetask/message"
)

type UpstreamHandler interface {
	// Logger returns the upstream handler logger.
	Logger() logr.Logger

	// ConvToNodeTask converts the upstream message to node task.
	ConvToNodeTask(nodename string, upmsg *taskmsg.UpstreamMessage) (wrap.NodeJobTask, error)

	// ReleaseExecutorConcurrent releases the executor concurrent when the node task is the final action.
	ReleaseExecutorConcurrent(res taskmsg.Resource) error

	// UpdateNodeTaskStatus updates the status of node task.
	UpdateNodeTaskStatus(jobname string, nodetask wrap.NodeJobTask) error
}

// upstreamHandlers is the map of upstream handlers.
var upstreamHandlers = make(map[string]UpstreamHandler)

// Init registers the upstream handlers.
func Init(ctx context.Context) {
	upstreamHandlers[operationsv1alpha2.ResourceImagePrePullJob] = newImagePrePullJobHandler(ctx)
}

// Start starts the upstream handler.
func Start(ctx context.Context, statusChan <-chan model.Message) {
	go func() {
		for {
			select {
			case <-ctx.Done():
				klog.Info("stop watching upstream messages of node task")
				return
			case msg, ok := <-statusChan:
				if !ok {
					klog.Info("the upstream status channel has been closed")
					return
				}
				data, err := msg.GetContentData()
				if err != nil {
					klog.Warningf("failed to get upstream content data, err: %v", err)
					continue
				}
				var upmsg taskmsg.UpstreamMessage
				if err := json.Unmarshal(data, &upmsg); err != nil {
					klog.Warningf("failed to unmarshal upstream message, err: %v", err)
					continue
				}
				res := taskmsg.ParseResource(msg.GetResource())
				handler, ok := upstreamHandlers[res.ResourceType]
				if !ok {
					klog.Warningf("invalid node task resource type %s", res.ResourceType)
					continue
				}
				if err := updateNodeJobTaskStatus(res, upmsg, handler); err != nil {
					handler.Logger().Error(err, "failed to update node task status",
						"job name", res.JobName, "node name", res.NodeName)
					continue
				}
			}
		}
	}()
}

// updateNodeJobTaskStatus updates the status of node job task.
func updateNodeJobTaskStatus(res taskmsg.Resource,
	upmsg taskmsg.UpstreamMessage, handler UpstreamHandler,
) error {
	nodetask, err := handler.ConvToNodeTask(res.NodeName, &upmsg)
	if err != nil {
		return fmt.Errorf("failed to set node task status, err: %v", err)
	}
	action, err := nodetask.Action()
	if err != nil {
		return err // Enough error messages.
	}
	if action.IsFinal() {
		if nodetask.Phase() == operationsv1alpha2.NodeTaskPhaseInProgress {
			nodetask.ToSuccessful()
		}
		if err := handler.ReleaseExecutorConcurrent(res); err != nil {
			// This error does not affect the process, just logger.
			handler.Logger().Error(err, "failed to release executor concurrent",
				"job name", res.JobName, "node name", res.NodeName)
		}
	} else {
		// The cloud actively transfers to the next action, which can reduce the number of reports from the edge.
		if next := action.Next(upmsg.Succ); next != nil {
			nodetask.SetAction(next)
		}
	}
	if err := handler.UpdateNodeTaskStatus(res.JobName, nodetask); err != nil {
		return err // Enough error messages.
	}
	return nil
}
