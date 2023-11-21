package httpserver

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/emicklei/go-restful"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/klog/v2"

	beehiveContext "github.com/kubeedge/beehive/pkg/core/context"
	beehiveModel "github.com/kubeedge/beehive/pkg/core/model"
	"github.com/kubeedge/kubeedge/cloud/pkg/common/modules"
	"github.com/kubeedge/kubeedge/cloud/pkg/nodeupgradejobcontroller/controller"
	commontypes "github.com/kubeedge/kubeedge/common/types"
)

const (
	millionByte = int64(3 * 1024 * 1024)
)

// reportTaskStatus report the status of task
func reportTaskStatus(request *restful.Request, response *restful.Response) {
	resp := commontypes.TaskStatus{}

	taskID := request.PathParameter("taskID")
	nodeID := request.PathParameter("nodeID")

	lr := &io.LimitedReader{
		R: request.Request.Body,
		N: millionByte + 1,
	}
	body, err := io.ReadAll(lr)
	if err != nil {
		response.WriteError(http.StatusBadRequest, fmt.Errorf("failed to get req body: %v", err))
		return
	}
	if lr.N <= 0 {
		response.WriteError(http.StatusBadRequest, errors.NewRequestEntityTooLargeError(fmt.Sprintf("the request body can only be up to 1MB in size")))
		return
	}
	if err = json.Unmarshal(body, &resp); err != nil {
		response.WriteError(http.StatusBadRequest, fmt.Errorf("failed to marshal task info: %v", err))
		return
	}

	//TODO The resource operation should be refactored, like: {type}/task/{taskID}/node/{nodeId}
	msg := beehiveModel.NewMessage("").SetRoute(modules.CloudHubModuleName, modules.CloudHubModuleGroup).
		SetResourceOperation(fmt.Sprintf("%s/%s/node/%s", resp.Type, taskID, nodeID), controller.NodeUpgrade).FillBody(resp)
	//TODO The message should be sent to the task manager.
	beehiveContext.Send(modules.NodeUpgradeJobControllerModuleName, *msg)

	if _, err = response.Write([]byte("ok")); err != nil {
		klog.Errorf("failed to send the task resp to edge , err: %v", err)
	}
}
