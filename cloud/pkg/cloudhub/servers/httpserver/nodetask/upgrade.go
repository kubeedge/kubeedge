/*
Copyright 2022 The KubeEdge Authors.
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
package nodetask

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/emicklei/go-restful"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/klog/v2"

	api "github.com/kubeedge/api/apis/fsm/v1alpha1"
	beehiveContext "github.com/kubeedge/beehive/pkg/core/context"
	beehiveModel "github.com/kubeedge/beehive/pkg/core/model"
	"github.com/kubeedge/kubeedge/cloud/pkg/common/modules"
	"github.com/kubeedge/kubeedge/cloud/pkg/taskmanager/util"
	commontypes "github.com/kubeedge/kubeedge/common/types"
)

const (
	UpgradeSuccess               = "upgrade_success"
	UpgradeFailedRollbackSuccess = "upgrade_failed_rollback_success"
	UpgradeFailedRollbackFailed  = "upgrade_failed_rollback_failed"
)

// UpgradeEdge upgrade the edgecore version
func UpgradeEdge(request *restful.Request, response *restful.Response) {
	resp := commontypes.NodeUpgradeJobResponse{}

	taskID := resp.UpgradeID
	taskType := util.TaskUpgrade
	nodeID := resp.NodeName

	lr := &io.LimitedReader{
		R: request.Request.Body,
		N: millionByte + 1,
	}
	body, err := io.ReadAll(lr)
	if err != nil {
		err = response.WriteError(http.StatusBadRequest, fmt.Errorf("failed to get req body: %v", err))
		if err != nil {
			klog.Warning(err.Error())
		}
		return
	}
	if lr.N <= 0 {
		err = response.WriteError(http.StatusBadRequest, errors.NewRequestEntityTooLargeError("the request body can only be up to 1MB in size"))
		if err != nil {
			klog.Warning(err.Error())
		}
		return
	}
	if err = json.Unmarshal(body, &resp); err != nil {
		err = response.WriteError(http.StatusBadRequest, fmt.Errorf("failed to marshal task info: %v", err))
		if err != nil {
			klog.Warning(err.Error())
		}
		return
	}
	newResp := commontypes.NodeTaskResponse{
		NodeName: resp.NodeName,
		Event:    "Upgrade",
		Action:   api.ActionSuccess,
		Reason:   resp.Reason,
	}

	if resp.Status == UpgradeFailedRollbackSuccess {
		newResp.Event = "Rollback"
		newResp.Action = api.ActionFailure
	}

	if resp.Status == UpgradeFailedRollbackFailed {
		newResp.Event = "Rollback"
		newResp.Action = api.ActionSuccess
	}

	msg := beehiveModel.NewMessage("").SetRoute(modules.CloudHubModuleName, modules.CloudHubModuleGroup).
		SetResourceOperation(fmt.Sprintf("task/%s/node/%s", taskID, nodeID), taskType).FillBody(newResp)
	beehiveContext.Send(modules.TaskManagerModuleName, *msg)

	if _, err = response.Write([]byte("ok")); err != nil {
		klog.Errorf("failed to send the task resp to edge , err: %v", err)
	}
}
