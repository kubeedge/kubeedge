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

	"github.com/go-logr/logr"
	"k8s.io/klog/v2"

	operationsv1alpha2 "github.com/kubeedge/api/apis/operations/v1alpha2"
	"github.com/kubeedge/beehive/pkg/core/model"
	"github.com/kubeedge/kubeedge/cloud/pkg/taskmanager/v1alpha2/executor"
	"github.com/kubeedge/kubeedge/pkg/nodetask/actionflow"
	taskmsg "github.com/kubeedge/kubeedge/pkg/nodetask/message"
)

type UpstreamHandler interface {
	// Logger returns the upstream handler logger.
	Logger() logr.Logger

	// GetAction returns the queried action of the node task.
	GetAction(name string) *actionflow.Action

	// UpdateNodeTaskStatus updates the status of node task.
	UpdateNodeTaskStatus(jobName, nodeName string, isFinalAction bool, upmsg taskmsg.UpstreamMessage) error
}

// upstreamHandlers is the map of upstream handlers.
var upstreamHandlers = make(map[string]UpstreamHandler)

// Init registers the upstream handlers.
func Init(ctx context.Context) {
	upstreamHandlers[operationsv1alpha2.ResourceNodeUpgradeJob] = newNodeUpgradeJobHandler(ctx)
	upstreamHandlers[operationsv1alpha2.ResourceImagePrePullJob] = newImagePrePullJobHandler(ctx)
	upstreamHandlers[operationsv1alpha2.ResourceConfigUpdateJob] = newConfigUpdateJobHandler(ctx)
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

				upmsg, res, err := parseUpstreamMessage(msg)
				if err != nil {
					klog.Errorf("failed to parse node task upstream message, err: %v", err)
					continue
				}

				// Get handler
				handler, ok := upstreamHandlers[res.ResourceType]
				if !ok {
					klog.Errorf("invalid node task resource type %s", res.ResourceType)
					continue
				}

				if err := handleUpstreamMessage(handler, upmsg, res); err != nil {
					handler.Logger().Error(err, "failed to update node task status",
						"job name", res.JobName, "node name", res.NodeName)
					continue
				}
			}
		}
	}()
}

func parseUpstreamMessage(msg model.Message,
) (upmsg taskmsg.UpstreamMessage, res taskmsg.Resource, err error) {
	// Parse the message content to the upstream message,
	data, err := msg.GetContentData()
	if err != nil {
		err = fmt.Errorf("failed to get upstream content data, err: %v", err)
		return
	}
	if err = json.Unmarshal(data, &upmsg); err != nil {
		err = fmt.Errorf("failed to unmarshal upstream message, err: %v", err)
		return
	}
	// parse the message resoure.
	res = taskmsg.ParseResource(msg.GetResource())
	return
}

func handleUpstreamMessage(
	handler UpstreamHandler,
	upmsg taskmsg.UpstreamMessage,
	res taskmsg.Resource,
) error {
	action := handler.GetAction(upmsg.Action)
	if action == nil {
		return fmt.Errorf("invalid %s action %s", res.ResourceType, upmsg.Action)
	}
	isFinalAction := upmsg.Succ && action.NextSuccessful == nil ||
		!upmsg.Succ && action.NextFailure == nil

	// It's the final action, so release the executor.
	if isFinalAction {
		if err := releaseExecutorConcurrent(res); err != nil {
			return fmt.Errorf("failed to release executor concurrent, err: %v", err)
		}
	}

	if err := handler.UpdateNodeTaskStatus(res.JobName, res.NodeName, isFinalAction, upmsg); err != nil {
		return fmt.Errorf("failed to update node task status, err: %v", err)
	}
	return nil
}

func releaseExecutorConcurrent(res taskmsg.Resource) error {
	exec, err := executor.GetExecutor(res.ResourceType, res.JobName)
	if err != nil && !errors.Is(err, executor.ErrExecutorNotExists) {
		return fmt.Errorf("failed to get executor, err: %v", err)
	}
	if exec != nil {
		exec.FinishTask()
	}
	return nil
}
