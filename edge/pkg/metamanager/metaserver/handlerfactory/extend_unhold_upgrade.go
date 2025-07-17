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
	"fmt"
	"net/http"

	beehiveContext "github.com/kubeedge/beehive/pkg/core/context"
	"github.com/kubeedge/beehive/pkg/core/model"
	"github.com/kubeedge/kubeedge/edge/pkg/common/modules"
	"k8s.io/klog/v2"
)

func (f *Factory) UnholdUpgrade() http.Handler {
	h := http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		ctx := req.Context()
		logger := klog.FromContext(ctx).WithName("unholdUpgrade")
		logger.V(1).Info("start to unhold upgrade")

		keyBytes, err := limitedReadBody(req, int64(3*1024*1024))
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		resource := fmt.Sprintf("namespace/%s/id", model.UnholdPodUpgrade)
		msg := model.NewMessage("").
			BuildRouter(modules.MetaManagerModuleName, "", resource, model.UpdateOperation).
			FillBody(keyBytes)

		beehiveContext.Send(modules.EdgedModuleName, *msg)

		w.WriteHeader(http.StatusOK)
	})
	return h
}
