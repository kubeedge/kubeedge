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
	"github.com/kubeedge/kubeedge/pkg/apis/operations/v1alpha1"
)

const (
	millionByte = int64(3 * 1024 * 1024)
)

// reportTaskStatus report the status of task
func reportTaskStatus(request *restful.Request, response *restful.Response) {
	resp := v1alpha1.TaskStatus{}
	taskID := request.PathParameter("taskID")
	taskType := request.PathParameter("taskType")
	nodeID := request.PathParameter("nodeID")

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

	msg := beehiveModel.NewMessage("").SetRoute(modules.CloudHubModuleName, modules.CloudHubModuleGroup).
		SetResourceOperation(fmt.Sprintf("task/%s/node/%s", taskID, nodeID), taskType).FillBody(resp)
	beehiveContext.Send(modules.TaskManagerModuleName, *msg)

	if _, err = response.Write([]byte("ok")); err != nil {
		klog.Errorf("failed to send the task resp to edge , err: %v", err)
	}
}
