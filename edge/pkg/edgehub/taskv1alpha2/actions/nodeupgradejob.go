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

package actions

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/go-logr/logr"
	"k8s.io/klog/v2"

	operationsv1alpha2 "github.com/kubeedge/api/apis/operations/v1alpha2"
	"github.com/kubeedge/kubeedge/edge/pkg/common/message"
	"github.com/kubeedge/kubeedge/pkg/nodetask/actionflow"
	taskmsg "github.com/kubeedge/kubeedge/pkg/nodetask/message"
)

func newNodeUpgradeJobRunner() *ActionRunner {
	logger := klog.Background().WithName("node-upgrade-job-runner")
	funcs := nodeUpgradeJobFuncs{
		logger: logger,
	}
	runner := &ActionRunner{
		Flow:               actionflow.FlowNodeUpgradeJob,
		ReportActionStatus: funcs.reportActionStatus,
		GetSpecSerializer:  funcs.getSpecSerializer,
		Logger:             logger,
	}
	runner.addAction(string(operationsv1alpha2.NodeUpgradeJobActionCheck), funcs.checkItems)
	runner.addAction(string(operationsv1alpha2.NodeUpgradeJobActionWaitingConfirmation), funcs.waitingConfirmation)
	runner.addAction(string(operationsv1alpha2.NodeUpgradeJobActionConfirm), funcs.confirm)
	return runner
}

type nodeUpgradeJobActionResponse struct {
	baseActionResponse
}

// nodeUpgradeJobFuncs used to control function scope
type nodeUpgradeJobFuncs struct {
	logger logr.Logger
}

func (nodeUpgradeJobFuncs) checkItems(_ctx context.Context, specser SpecSerializer) ActionResponse {
	resp := new(nodeUpgradeJobActionResponse)
	spec, ok := specser.GetSpec().(*operationsv1alpha2.NodeUpgradeJobSpec)
	if !ok {
		resp.err = fmt.Errorf("failed to conv spec to NodeUpgradeJobSpec, actual type %T", specser.GetSpec())
		return resp
	}
	if err := PreCheck(spec.CheckItems); err != nil {
		resp.err = err
		return resp
	}
	resp.doNext = true
	return resp
}

func (nodeUpgradeJobFuncs) waitingConfirmation(_ctx context.Context, specser SpecSerializer) ActionResponse {
	resp := new(nodeUpgradeJobActionResponse)
	spec, ok := specser.GetSpec().(*operationsv1alpha2.NodeUpgradeJobSpec)
	if !ok {
		resp.err = fmt.Errorf("failed to conv spec to NodeUpgradeJobSpec, actual type %T", specser.GetSpec())
		return resp
	}
	// If confirmation is required, return false to block the action flow.
	resp.doNext = !spec.RequireConfirmation
	return resp
}

func (nodeUpgradeJobFuncs) confirm(_ctx context.Context, _specser SpecSerializer) ActionResponse {
	// Used to process the confirmation action and transition to backup.
	return &nodeUpgradeJobActionResponse{
		baseActionResponse: baseActionResponse{doNext: true},
	}
}

func (nodeUpgradeJobFuncs) backup(_ctx context.Context, _specser SpecSerializer) ActionResponse {
	resp := new(nodeUpgradeJobActionResponse)
	// TODO: ..
	return resp
}

func (nodeUpgradeJobFuncs) getSpecSerializer(specData []byte) (SpecSerializer, error) {
	return NewSpecSerializer(specData, func(d []byte) (any, error) {
		var spec operationsv1alpha2.NodeUpgradeJobSpec
		if err := json.Unmarshal(d, &spec); err != nil {
			return nil, err
		}
		return &spec, nil
	})
}

func (nodeUpgradeJobFuncs) reportActionStatus(jobname, nodename, action string, resp ActionResponse) {
	res := taskmsg.Resource{
		APIVersion:   operationsv1alpha2.SchemeGroupVersion.String(),
		ResourceType: operationsv1alpha2.ResourceNodeUpgradeJob,
		JobName:      jobname,
		NodeName:     nodename,
	}
	body := taskmsg.UpstreamMessage{
		Action: action,
	}
	if err := resp.Error(); err != nil {
		body.Succ = false
		body.Reason = err.Error()
	} else {
		body.Succ = true
	}
	message.ReportNodeTaskStatus(res, body)
}
