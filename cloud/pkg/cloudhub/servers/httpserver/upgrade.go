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
	"net/http"

	"k8s.io/klog/v2"

	beehiveContext "github.com/kubeedge/beehive/pkg/core/context"
	beehiveModel "github.com/kubeedge/beehive/pkg/core/model"
	"github.com/kubeedge/kubeedge/cloud/pkg/common/modules"
	commontypes "github.com/kubeedge/kubeedge/common/types"
)

// upgradeEdge upgrade the edgecore version
func upgradeEdge(w http.ResponseWriter, r *http.Request) {
	klog.Infof("DEBUG DEBUG receive upgrade msg")
	resp := commontypes.UpgradeResponse{}

	defer func() {
		if _, err := w.Write([]byte("ok")); err != nil {
			klog.Errorf("failed to send upgrade edge resp, err: %v", err)
		}
	}()

	body, err := io.ReadAll(r.Body)
	if err != nil {
		klog.Errorf("failed to get req body: %v", err)
		return
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		klog.Errorf("failed to marshal upgrade info: %v", err)
		return
	}

	msg := beehiveModel.NewMessage("").SetRoute(modules.CloudHubModuleName, modules.CloudHubModuleName).
		SetResourceOperation(fmt.Sprintf("upgrade/%s/node/%s", resp.UpgradeID, resp.NodeName), "upgrade").FillBody(resp)
	beehiveContext.Send(modules.UpgradeControllerModuleName, *msg)
	if err != nil {
		klog.Errorf("send upgrade resp msg failed: %v", err)
		return
	}
}
