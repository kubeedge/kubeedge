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
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"k8s.io/klog/v2"

	operationsv1alpha2 "github.com/kubeedge/api/apis/operations/v1alpha2"
	"github.com/kubeedge/kubeedge/common/types"
	commontypes "github.com/kubeedge/kubeedge/common/types"
	"github.com/kubeedge/kubeedge/edge/cmd/edgecore/app/options"
	commonmsg "github.com/kubeedge/kubeedge/edge/pkg/common/message"
	"github.com/kubeedge/kubeedge/edge/pkg/metamanager/dao/dbclient"
	"github.com/kubeedge/kubeedge/edge/pkg/taskmanager/actions"
	"github.com/kubeedge/kubeedge/edge/pkg/taskmanager/v1alpha1/taskexecutor"
)

func (f *Factory) ConfirmUpgrade() http.Handler {
	h := http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		ctx := req.Context()
		logger := klog.FromContext(ctx).WithName("confirmUpgrade")
		logger.V(1).Info("start to confirm upgrade")

		upgradeDao := dbclient.NewUpgrade()
		jobname, nodename, spec, err := upgradeDao.Get()
		if err != nil {
			http.Error(w, fmt.Sprintf("failed to get upgrade spec, err: %v", err),
				http.StatusInternalServerError)
			return
		}
		// When the v1alpha1 upgrade task is no longer supported, The logical expression needs to be modified to:
		// 	if spec != nil {
		// 		...
		// 	} else {
		// 		http.Error(w, "there are no valid upgrade tasks to execute", http.StatusBadRequest)
		// 		return
		// 	}
		if spec != nil { // v1alpha2+
			logger.V(1).Info("execute v1alpha2+ upgrade task")
			// The confirmation operation is only responsible for activating the upgrade,
			// so this API does not need to be aware of errors caused by the upgrade.
			go func() {
				if err := doUpgrade(jobname, nodename, spec); err != nil {
					logger.Error(err, "failed to execute v1alpha2+ upgrade task")
				}
			}()
		} else { // v1alpha1
			upgradeV1alpha1Dao := dbclient.NewUpgradeV1alpha1()
			upgradeReq, err := upgradeV1alpha1Dao.Get()
			if err != nil {
				http.Error(w, fmt.Sprintf("failed to get upgrade request, err: %v", err),
					http.StatusInternalServerError)
				return
			}
			if upgradeReq == nil {
				http.Error(w, "there are no valid upgrade tasks to execute", http.StatusBadRequest)
				return
			}
			logger.V(1).Info("execute v1alpha1 upgrade task")
			go func() {
				if err := doV1alpha1Upgrade(upgradeReq); err != nil {
					logger.Error(err, "failed to execute v1alpha1 upgrade task")
				}
			}()
		}
		w.WriteHeader(http.StatusOK)
	})
	return h
}

func doUpgrade(jobname, nodename string, spec *operationsv1alpha2.NodeUpgradeJobSpec) error {
	runner := actions.GetRunner(operationsv1alpha2.ResourceNodeUpgradeJob)
	if runner == nil {
		return fmt.Errorf("invalid resource type %s", operationsv1alpha2.ResourceNodeUpgradeJob)
	}
	specData, err := json.Marshal(spec)
	if err != nil {
		return fmt.Errorf("failed to marshal NodeUpgradeJobSpec to json, err: %v", err)
	}
	runner.RunAction(context.Background(), jobname, nodename,
		string(operationsv1alpha2.NodeUpgradeJobActionBackUp), specData)
	return nil
}

func doV1alpha1Upgrade(req *types.NodeTaskRequest) error {
	executor, err := taskexecutor.GetExecutor(taskexecutor.TaskUpgrade)
	if err != nil {
		return fmt.Errorf("failed to get v1alpha1 upgrade task executor, err: %v", err)
	}
	event, err := executor.Do(*req)
	if err != nil {
		return fmt.Errorf("failed to execute v1alpha1 upgrade task, err: %v", err)
	}
	// Report the result of the task to the cloud.
	resp := commontypes.NodeTaskResponse{
		NodeName: options.GetEdgeCoreConfig().Modules.Edged.HostnameOverride,
		Event:    event.Type,
		Action:   event.Action,
		Reason:   event.Msg,
	}
	commonmsg.ReportTaskResult(req.Type, req.TaskID, resp)
	return nil
}
