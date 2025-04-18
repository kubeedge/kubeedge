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

package handlerfactory

import (
	"net/http"

	"k8s.io/klog/v2"

	"github.com/kubeedge/kubeedge/common/types"
	commontypes "github.com/kubeedge/kubeedge/common/types"
	"github.com/kubeedge/kubeedge/edge/cmd/edgecore/app/options"
	commonmsg "github.com/kubeedge/kubeedge/edge/pkg/common/message"
	daov2 "github.com/kubeedge/kubeedge/edge/pkg/metamanager/dao/v2"
	"github.com/kubeedge/kubeedge/edge/pkg/taskmanager/v1alpha1/taskexecutor"
)

func (f *Factory) ConfirmUpgrade() http.Handler {
	h := http.HandlerFunc(func(w http.ResponseWriter, _req *http.Request) {
		var nodeTaskReq types.NodeTaskRequest
		upgradev1alpha1 := daov2.NewUpgradeV1alpha1()
		req, err := upgradev1alpha1.Get()
		if err != nil {
			// TODO: hendle error
			// http.Error(w, fmt.Sprintf("run upgrade command %s failed: %v, res: %s", command, err, s), http.StatusInternalServerError)
		}
		if req != nil {
			executor, err := taskexecutor.GetExecutor(taskexecutor.TaskUpgrade)
			if err != nil {
				// TODO: hendle error
			}
			event, err := executor.Do(nodeTaskReq)
			if err != nil {
				// TODO: hendle error
			}
			defer func() {
				if err := upgradev1alpha1.Delete(); err != nil {
					klog.Warningf("failed to delete v1alpha1 upgrade task: %v", err)
				}
			}()
			resp := commontypes.NodeTaskResponse{
				NodeName: options.GetEdgeCoreConfig().Modules.Edged.HostnameOverride,
				Event:    event.Type,
				Action:   event.Action,
				Reason:   event.Msg,
			}
			commonmsg.ReportTaskResult(req.Type, req.TaskID, resp)
		}
	})
	return h
}
