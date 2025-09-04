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
	"strings"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/klog/v2"

	beehiveContext "github.com/kubeedge/beehive/pkg/core/context"
	"github.com/kubeedge/beehive/pkg/core/model"
	"github.com/kubeedge/kubeedge/edge/pkg/common/modules"
	"github.com/kubeedge/kubeedge/keadm/cmd/keadm/app/cmd/util/metaclient"
)

func (f *Factory) UnholdUpgrade() http.Handler {
	h := http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		ctx := req.Context()
		logger := klog.FromContext(ctx).WithName("unholdUpgrade")
		logger.V(4).Info("start to unhold upgrade")

		keyBytes, err := limitedReadBody(req, int64(3*1024*1024))
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		key := string(keyBytes)

		parts := strings.SplitN(key, "/", 2)
		if len(parts) != 2 {
			http.Error(w, "invalid format, expected <namespace>/<name>", http.StatusBadRequest)
			return
		}
		namespace, name := parts[0], parts[1]

		// use kubeclient to get pod metadata
		clientset, err := metaclient.KubeClient()
		if err != nil {
			http.Error(w, fmt.Sprintf("cannot init kube client: %v", err), http.StatusInternalServerError)
			return
		}

		pod, err := clientset.CoreV1().Pods(namespace).Get(ctx, name, metav1.GetOptions{})
		if err != nil {
			http.Error(w, fmt.Sprintf("pod not found: %v", err), http.StatusNotFound)
			return
		}

		// validate pod annotation and status
		if pod.Annotations["edge.kubeedge.io/hold-upgrade"] != "true" {
			http.Error(w, "pod is not marked with hold-upgrade annotation", http.StatusBadRequest)
			return
		}

		if pod.Status.Phase != v1.PodPending {
			http.Error(w, "pod is not in pending phase", http.StatusBadRequest)
			return
		}

		// resource := fmt.Sprintf("namespace/%s/pod/%s", namespace, name)
		resource := fmt.Sprintf("%s/pod/%s", namespace, name)
		msg := model.NewMessage("").
			BuildRouter(modules.MetaManagerModuleName, "", resource, model.UnholdUpgradeOperation)
		beehiveContext.Send(modules.EdgedModuleName, *msg)

		w.WriteHeader(http.StatusOK)
	})
	return h
}
