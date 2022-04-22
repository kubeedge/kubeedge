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

package httpserver

import (
	"encoding/json"
	"fmt"
	"io"

	"github.com/emicklei/go-restful"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/klog/v2"

	beehiveContext "github.com/kubeedge/beehive/pkg/core/context"
	beehiveModel "github.com/kubeedge/beehive/pkg/core/model"
	"github.com/kubeedge/kubeedge/cloud/pkg/common/modules"
	"github.com/kubeedge/kubeedge/cloud/pkg/nodeupgradejobcontroller/controller"
	commontypes "github.com/kubeedge/kubeedge/common/types"
)

// upgradeEdge upgrade the edgecore version
func upgradeEdge(request *restful.Request, response *restful.Response) {
	resp := commontypes.NodeUpgradeJobResponse{}

	defer func() {
		if _, err := response.Write([]byte("ok")); err != nil {
			klog.Errorf("failed to send upgrade edge resp, err: %v", err)
		}
	}()

	limit := int64(3 * 1024 * 1024)
	lr := &io.LimitedReader{
		R: request.Request.Body,
		N: limit + 1,
	}
	body, err := io.ReadAll(lr)
	if err != nil {
		klog.Errorf("failed to get req body: %v", err)
		return
	}
	if lr.N <= 0 {
		klog.Errorf("%v", errors.NewRequestEntityTooLargeError(fmt.Sprintf("limit is %d", limit)))
		return
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		klog.Errorf("failed to marshal upgrade info: %v", err)
		return
	}

	msg := beehiveModel.NewMessage("").SetRoute(modules.CloudHubModuleName, modules.CloudHubModuleName).
		SetResourceOperation(fmt.Sprintf("%s/%s/node/%s", controller.NodeUpgrade, resp.UpgradeID, resp.NodeName), controller.NodeUpgrade).FillBody(resp)
	beehiveContext.Send(modules.NodeUpgradeJobControllerModuleName, *msg)
}
